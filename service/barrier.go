/*
 * Copyright 2021 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/shareds"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
	"golang.org/x/sync/singleflight"
	"time"
)

type Barrier interface {
	Do(ctx context.Context, key string, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError)
	Forget(ctx context.Context, key string)
}

func defaultBarrier() Barrier {
	return &sfgBarrier{
		group: singleflight.Group{},
	}
}

type sfgBarrier struct {
	group singleflight.Group
}

func (barrier *sfgBarrier) Do(_ context.Context, key string, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError) {
	var doErr error
	result, doErr, _ = barrier.group.Do(key, func() (interface{}, error) {
		return fn()
	})
	if doErr != nil {
		err = errors.Map(doErr)
	}
	return
}

func (barrier *sfgBarrier) Forget(_ context.Context, key string) {
	barrier.group.Forget(key)
}

// todo mv into shared , and redis shared barrier use lua to implement，same as shared locker
// 大致逻辑：注意redis开启多线程
// setNX，当结果为0则已存在，进入wait，直到status存在，返回status和value，如果wait期间key不存在，则进入起始状态
// 起始: set key = hash{status:0, value: nil}，然后lua结束，返回1，执行逻辑，成功结果或status=1，value=结果，错误则status=-1，value=错误，expire 10s
// 后续: 进入wait
// lua的wait，使用 repeat until
/*
function wait(key)
	status = 0 // 当key不存在时，status为-2
    repeat
		// 取key
	until( status != 0 )
end
*/
func clusterBarrier(shared shareds.Shared, resultTTL time.Duration) Barrier {
	return &sharedBarrier{
		group:     singleflight.Group{},
		shared:    shared,
		resultTTL: resultTTL,
	}
}

type sharedBarrier struct {
	group     singleflight.Group
	shared    shareds.Shared
	resultTTL time.Duration
}

func (barrier *sharedBarrier) Do(ctx context.Context, key string, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError) {
	var doErr error
	result, doErr, _ = barrier.group.Do(key, func() (v interface{}, err error) {
		barrierKey := bytex.FromString(fmt.Sprintf("fns/barrier/%s", key))
		locker, getLockerErr := barrier.shared.Lockers().Acquire(ctx, barrierKey, 2*time.Second)
		if getLockerErr != nil {
			err = errors.Warning("fns: shared barrier execute failed").WithCause(getLockerErr)
			return
		}
		lockErr := locker.Lock(ctx)
		if lockErr != nil {
			err = errors.Warning("fns: shared barrier execute failed").WithCause(lockErr)
			return
		}
		resultKey := bytex.FromString(fmt.Sprintf("fns/barrier/%s/result", key))
		cached, has, getErr := barrier.shared.Store().Get(ctx, resultKey)
		if getErr != nil {
			_ = locker.Unlock(ctx)
			err = errors.Warning("fns: shared barrier execute failed").WithCause(getErr)
			return
		}
		if has {
			if len(cached) < 2 {
				_ = locker.Unlock(ctx)
				err = errors.Warning("fns: shared barrier execute failed").WithCause(errors.Warning("cached value is out of control"))
				return
			}
			if binary.BigEndian.Uint16(cached[0:2]) == 1 {
				v = json.RawMessage(cached[1:])
			} else if binary.BigEndian.Uint16(cached[0:2]) == 2 {
				err = errors.Decode(cached[1:])
			} else {
				err = errors.Warning("fns: shared barrier execute failed").WithCause(errors.Warning("cached value is out of control"))
			}
			_ = locker.Unlock(ctx)
			return
		}
		rb := bytebufferpool.Get()
		v, err = fn()
		if err != nil {
			p, _ := json.Marshal(err)
			head := make([]byte, 2)
			binary.BigEndian.PutUint16(head, 2)
			_, _ = rb.Write(head)
			_, _ = rb.Write(p)
		} else {
			p, encodeErr := json.Marshal(v)
			if encodeErr != nil {
				p, _ = json.Marshal(errors.Warning("fns: encode result failed").WithCause(encodeErr))
				head := make([]byte, 2)
				binary.BigEndian.PutUint16(head, 2)
				_, _ = rb.Write(head)
				_, _ = rb.Write(p)
			} else {
				head := make([]byte, 2)
				binary.BigEndian.PutUint16(head, 1)
				_, _ = rb.Write(head)
				_, _ = rb.Write(p)
			}
		}
		cached = rb.Bytes()
		bytebufferpool.Put(rb)
		_ = barrier.shared.Store().SetWithTTL(ctx, resultKey, cached, barrier.resultTTL)
		_ = locker.Unlock(ctx)
		return
	})
	if doErr != nil {
		err = errors.Map(doErr)
	}
	return
}

func (barrier *sharedBarrier) Forget(ctx context.Context, key string) {
	barrier.group.Forget(key)
	barrierKey := bytex.FromString(fmt.Sprintf("fns/barrier/%s", key))
	_ = barrier.shared.Store().Remove(ctx, barrierKey)
	resultKey := bytex.FromString(fmt.Sprintf("fns/barrier/%s/result", key))
	_ = barrier.shared.Store().Remove(ctx, resultKey)
}

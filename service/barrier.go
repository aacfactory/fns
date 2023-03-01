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
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/shared"
	"github.com/aacfactory/json"
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

func clusterBarrier(store shared.Store, lockers shared.Lockers, resultTTL time.Duration) Barrier {
	return &sharedBarrier{
		group:     singleflight.Group{},
		lockers:   lockers,
		store:     store,
		resultTTL: resultTTL,
	}
}

type sharedBarrier struct {
	group     singleflight.Group
	lockers   shared.Lockers
	store     shared.Store
	resultTTL time.Duration
}

func (barrier *sharedBarrier) Do(ctx context.Context, key string, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError) {
	var doErr error
	result, doErr, _ = barrier.group.Do(key, func() (v interface{}, err error) {
		barrierKey := bytex.FromString(fmt.Sprintf("fns_barrier/%s", key))
		locker, lockErr := barrier.lockers.Lock(ctx, barrierKey, 2*time.Second)
		if lockErr != nil {
			err = errors.Warning("fns: shared barrier execute failed").WithCause(lockErr)
			return
		}
		defer locker.Unlock()
		resultKey := bytex.FromString(fmt.Sprintf("fns_barrier/%s/result", key))
		cached, has, getErr := barrier.store.Get(ctx, resultKey)
		if getErr != nil {
			err = errors.Warning("fns: shared barrier execute failed").WithCause(getErr)
			return
		}
		if has {
			if cached[0] == '1' {
				v = json.RawMessage(cached[1:])
			} else {
				err = errors.Decode(cached[1:])
			}
			return
		}
		v, err = fn()
		if err != nil {
			p, _ := json.Marshal(err)
			cached = bytes.Join([][]byte{{'0'}, p}, []byte{})
		} else {
			p, _ := json.Marshal(v)
			cached = bytes.Join([][]byte{{'1'}, p}, []byte{})
		}
		_ = barrier.store.SetWithTTL(ctx, resultKey, cached, barrier.resultTTL)
		return
	})
	if doErr != nil {
		err = errors.Map(doErr)
	}
	return
}

func (barrier *sharedBarrier) Forget(ctx context.Context, key string) {
	barrier.group.Forget(key)
	barrierKey := bytex.FromString(fmt.Sprintf("fns_barrier/%s", key))
	_ = barrier.store.Remove(ctx, barrierKey)
	resultKey := bytex.FromString(fmt.Sprintf("fns_barrier/%s/result", key))
	_ = barrier.store.Remove(ctx, resultKey)
}

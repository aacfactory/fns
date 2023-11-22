/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package clusters

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/json"
	"golang.org/x/sync/singleflight"
	"time"
)

var (
	prefix = []byte("fns/barrier/")
)

func NewBarrierValue() BarrierValue {
	p := make([]byte, 0, 1)
	return append(p, 'X')
}

type BarrierValue []byte

func (bv BarrierValue) Exist() bool {
	return len(bv) > 1
}

func (bv BarrierValue) Forgot() bool {
	return len(bv) > 1 && bv[0] == 'G' && bv[1] == 'G'
}

func (bv BarrierValue) Forget() BarrierValue {
	n := bv[:1]
	n[0] = 'G'
	return append(n, 'G')
}

func (bv BarrierValue) Value() (data []byte, err error) {
	if len(bv) < 2 {
		return
	}
	succeed := bv[0] == 'T'
	if succeed {
		if bv[1] == 'N' {
			return
		}
		data = bv[2:]
		return
	}
	if bv[1] == 'C' {
		err = errors.Decode(bv[2:])
	} else if bv[1] == 'S' {
		err = fmt.Errorf(bytex.ToString(bv[2:]))
	}
	return
}

func (bv BarrierValue) Succeed(v interface{}) (n BarrierValue, err error) {
	if v == nil {
		n = bv[:1]
		n[0] = 'T'
		n = append(n, 'N')
		return
	}
	p, encodeErr := json.Marshal(v)
	if encodeErr != nil {
		err = errors.Warning("fns: set succeed value into barrier value failed").WithCause(encodeErr)
		return
	}
	n = bv[:1]
	n = append(n, 'V')
	n = append(n, p...)
	return
}

func (bv BarrierValue) Failed(v error) (n BarrierValue) {
	n = bv[:1]
	n[0] = 'F'
	codeErr, ok := v.(errors.CodeError)
	if ok {
		n = append(n, 'C')
		p, _ := json.Marshal(codeErr)
		n = append(n, p...)
	} else {
		n = append(n, 'S')
		n = append(n, bytex.FromString(v.Error())...)
	}
	return
}

func NewBarrier(config BarrierConfig, shared shareds.Shared) (b barriers.Barrier) {
	ttl := config.TTL
	interval := config.Interval
	loops := 0
	if ttl < 1 {
		ttl = 10 * time.Second
	}
	if interval < 1 {
		interval = 100 * time.Millisecond
	}
	if interval >= ttl {
		loops = 10
		interval = ttl / time.Duration(loops)
	} else {
		loops = int(ttl / interval)
	}
	b = &Barrier{
		group:      new(singleflight.Group),
		standalone: config.Standalone,
		ttl:        ttl,
		interval:   interval,
		loops:      loops,
		store:      shared.Store(),
		lockers:    shared.Lockers(),
	}
	return
}

type BarrierConfig struct {
	TTL        time.Duration `json:"ttl"`
	Interval   time.Duration `json:"interval"`
	Standalone bool          `json:"standalone"`
}

type Barrier struct {
	group      *singleflight.Group
	standalone bool
	ttl        time.Duration
	interval   time.Duration
	loops      int
	store      shareds.Store
	lockers    shareds.Lockers
}

func (b *Barrier) Do(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (result barriers.Result, err error) {
	if len(key) == 0 {
		key = []byte{'-'}
	}

	r, doErr, _ := b.group.Do(bytex.ToString(key), func() (r interface{}, err error) {
		if b.standalone {
			r, err = fn()
			return
		}
		key = append(prefix, key...)
		r, err = b.doRemote(ctx, key, fn)
		return
	})
	if doErr != nil {
		err = errors.Map(doErr)
		return
	}
	result = scanner.New(r)
	return
}

func (b *Barrier) doRemote(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (r interface{}, err error) {
	locker, lockerErr := b.lockers.Acquire(ctx, key, b.ttl)
	if lockerErr != nil {
		err = errors.Warning("fns: barrier failed").WithCause(lockerErr)
		return
	}
	lockErr := locker.Lock(ctx)
	if lockErr != nil {
		err = errors.Warning("fns: barrier failed").WithCause(lockErr)
		return
	}

	value, has, getErr := b.store.Get(ctx, key)
	if getErr != nil {
		_ = locker.Unlock(ctx)
		err = errors.Warning("fns: barrier failed").WithCause(getErr)
		return
	}
	if !has {
		bv := NewBarrierValue()
		setErr := b.store.SetWithTTL(ctx, key, bv, b.ttl)
		if setErr != nil {
			_ = locker.Unlock(ctx)
			err = errors.Warning("fns: barrier failed").WithCause(setErr)
			return
		}
	}
	unlockErr := locker.Unlock(ctx)
	if unlockErr != nil {
		err = errors.Warning("fns: barrier failed").WithCause(unlockErr)
		return
	}

	if has {
		bv := BarrierValue(value)
		exist := false
		for i := 0; i < b.loops; i++ {
			if exist = bv.Exist(); exist {
				if bv.Forgot() {
					exist = false
					break
				}
				r, err = bv.Value()
				break
			}
			time.Sleep(b.interval)
		}
		if !exist {
			r, err = b.doRemote(ctx, key, fn)
		}
	} else {
		bv := NewBarrierValue()
		r, err = fn()
		if err != nil {
			bv.Failed(err)
		} else {
			bv, err = bv.Succeed(r)
			if err != nil {
				err = errors.Warning("fns: barrier failed").WithCause(err)
				return
			}
		}
		setErr := b.store.SetWithTTL(ctx, key, bv, b.ttl)
		if setErr != nil {
			err = errors.Warning("fns: barrier failed").WithCause(setErr)
			return
		}
	}

	return
}

func (b *Barrier) Forget(ctx context.Context, key []byte) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	b.group.Forget(bytex.ToString(key))
	if b.standalone {
		return
	}

	store := runtime.Load(ctx).Shared().Store()
	key = append(prefix, key...)
	_ = store.SetWithTTL(ctx, key, NewBarrierValue().Forget(), b.ttl)
}

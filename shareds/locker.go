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

package shareds

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrLockTimeout = fmt.Errorf("fns: shared lockers lock timeout failed")
)

type Locker interface {
	Lock(ctx context.Context) (err error)
	Unlock(ctx context.Context) (err error)
}

type Lockers interface {
	Acquire(ctx context.Context, key []byte, ttl time.Duration) (locker Locker, err error)
	Close()
}

type localLocker struct {
	key       []byte
	ttl       time.Duration
	mutex     sync.Locker
	releaseCh chan<- []byte
	done      chan struct{}
	locked    int64
}

func (locker *localLocker) Lock(ctx context.Context) (err error) {
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline && deadline.Before(time.Now().Add(locker.ttl)) {
		locker.ttl = deadline.Sub(time.Now())
	}
	if locker.ttl == 0 {
		atomic.StoreInt64(&locker.locked, 1)
		locker.mutex.Lock()
		return
	}
	go func(ctx context.Context, locker *localLocker) {
		atomic.StoreInt64(&locker.locked, 1)
		locker.mutex.Lock()
		locker.done <- struct{}{}
		close(locker.done)
	}(ctx, locker)
	select {
	case <-locker.done:
		break
	case <-time.After(locker.ttl):
		err = ErrLockTimeout
		break
	}
	if err != nil {
		_ = locker.Unlock(ctx)
	}
	return
}

func (locker *localLocker) Unlock(_ context.Context) (err error) {
	if atomic.LoadInt64(&locker.locked) > 0 {
		locker.mutex.Unlock()
		atomic.StoreInt64(&locker.locked, 0)
	}
	locker.releaseCh <- locker.key
	return
}

func LocalLockers() Lockers {
	v := &localLockers{
		mutex:     &sync.Mutex{},
		lockers:   make(map[string]*reuseLocker),
		releaseCh: make(chan []byte, 10240),
	}
	go v.listenRelease()
	return v
}

type localLockers struct {
	mutex     *sync.Mutex
	lockers   map[string]*reuseLocker
	releaseCh chan []byte
}

func (lockers *localLockers) Acquire(_ context.Context, key []byte, ttl time.Duration) (locker Locker, err error) {
	if key == nil || len(key) == 0 {
		err = fmt.Errorf("%+v", errors.Warning("fns: shared lockers acquire failed").WithCause(errors.Warning("key is required")))
		return
	}
	lockers.mutex.Lock()
	rl, has := lockers.lockers[bytex.ToString(key)]
	if !has {
		rl = &reuseLocker{
			mutex: &sync.Mutex{},
			times: 0,
		}
		lockers.lockers[bytex.ToString(key)] = rl
	}
	rl.times++
	locker = &localLocker{
		key:       key,
		ttl:       ttl,
		mutex:     rl.mutex,
		releaseCh: lockers.releaseCh,
		done:      make(chan struct{}, 1),
		locked:    0,
	}
	lockers.mutex.Unlock()
	return
}

func (lockers *localLockers) Close() {}

func (lockers *localLockers) listenRelease() {
	for {
		key, ok := <-lockers.releaseCh
		if !ok {
			break
		}
		lockers.mutex.Lock()
		v, has := lockers.lockers[bytex.ToString(key)]
		if !has {
			lockers.mutex.Unlock()
			continue
		}
		v.times--
		if v.times < 1 {
			delete(lockers.lockers, bytex.ToString(key))
		}
		lockers.mutex.Unlock()
	}
	return
}

type reuseLocker struct {
	mutex sync.Locker
	times int64
}

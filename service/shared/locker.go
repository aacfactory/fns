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

package shared

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"sync"
	"time"
)

type Locker interface {
	Unlock()
}

type Lockers interface {
	Lock(ctx context.Context, key []byte, ttl time.Duration) (locker Locker, err errors.CodeError)
}

type LocalSharedLocker struct {
	lockers *LocalLockers
	key     []byte
	mutex   sync.Locker
}

func (locker *LocalSharedLocker) Unlock() {
	locker.mutex.Unlock()
	locker.lockers.release(locker.key)
}

func NewLocalLockers() *LocalLockers {
	return &LocalLockers{
		mutex:   new(sync.Mutex),
		lockers: make(map[string]sync.Locker),
	}
}

type LocalLockers struct {
	mutex   *sync.Mutex
	lockers map[string]sync.Locker
}

func (lockers *LocalLockers) Lock(_ context.Context, key []byte, _ time.Duration) (locker Locker, err errors.CodeError) {
	lockers.mutex.Lock()
	defer lockers.mutex.Unlock()
	mKey := bytex.ToString(key)
	mutex, exist := lockers.lockers[mKey]
	if exist {
		lockers.lockers[mKey] = mutex
	} else {
		mutex = &sync.Mutex{}
		lockers.lockers[mKey] = mutex
	}
	mutex.Lock()
	locker = &LocalSharedLocker{
		lockers: lockers,
		key:     key,
		mutex:   mutex,
	}
	return
}

func (lockers *LocalLockers) release(key []byte) {
	lockers.mutex.Lock()
	defer lockers.mutex.Unlock()
	mKey := bytex.ToString(key)
	delete(lockers.lockers, mKey)
}

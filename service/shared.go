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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/dgraph-io/ristretto"
	"sync"
	"time"
)

type SharedStore interface {
	Set(key []byte, value []byte, timeout time.Duration) (err errors.CodeError)
	Get(key []byte) (value []byte, err error)
	Remove(key []byte) (err error)
}

type Shared interface {
	Store() (store SharedStore)
	Lock(key []byte, timeout time.Duration) (locker sync.Locker, err errors.CodeError)
	Close()
}

type InMemSharedStore struct {
	cache *ristretto.Cache
}

func (store *InMemSharedStore) Set(key []byte, value []byte, timeout time.Duration) (err errors.CodeError) {
	store.cache.SetWithTTL(key, value, int64(len(value)), timeout)
	return
}

func (store *InMemSharedStore) Get(key []byte) (value []byte, err error) {
	v, has := store.cache.Get(key)
	if !has {
		return
	}
	value, has = v.([]byte)
	if !has {
		err = errors.Warning("fns: shared store get failed").WithMeta("key", string(key)).WithCause(fmt.Errorf("value type is not matched"))
		return
	}
	return
}

func (store *InMemSharedStore) Remove(key []byte) (err error) {
	store.cache.Del(key)
	return
}

func newMemShared(memSize int64) (shared Shared, err error) {
	cache, cacheErr := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     memSize,
		BufferItems: 64,
	})
	if cacheErr != nil {
		err = errors.Warning("create local shared failed").WithCause(cacheErr)
		return
	}
	shared = &InMemShared{
		store: &InMemSharedStore{
			cache: cache,
		},
		mutex:   &sync.Mutex{},
		lockers: make(map[string]sync.Locker),
	}
	return
}

type InMemShared struct {
	store   *InMemSharedStore
	mutex   *sync.Mutex
	lockers map[string]sync.Locker
}

func (shared *InMemShared) Store() (store SharedStore) {
	store = shared.store
	return
}

func (shared *InMemShared) Lock(key []byte, _ time.Duration) (locker sync.Locker, err errors.CodeError) {
	shared.mutex.Lock()
	has := false
	locker, has = shared.lockers[string(key)]
	if !has {
		locker = &sync.Mutex{}
		shared.lockers[string(key)] = locker
	}
	shared.mutex.Unlock()
	return
}

func (shared *InMemShared) Close() {
	shared.store.cache.Close()
	return
}

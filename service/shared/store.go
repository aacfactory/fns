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
	"encoding/binary"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/container/smap"
	"github.com/aacfactory/fns/commons/mmhash"
	"github.com/dgraph-io/ristretto"
	"sync"
	"time"
)

type Store interface {
	Get(ctx context.Context, key []byte) (value []byte, has bool, err errors.CodeError)
	Set(ctx context.Context, key []byte, value []byte) (err errors.CodeError)
	SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err errors.CodeError)
	Incr(ctx context.Context, key []byte, delta int64) (v int64, err errors.CodeError)
	ExpireKey(ctx context.Context, key []byte, ttl time.Duration) (err errors.CodeError)
	Remove(ctx context.Context, key []byte) (err errors.CodeError)
	Close()
}

func LocalStore(memSize int64) (store Store, err error) {
	if memSize < 1 {
		memSize = 64 * bytex.MEGABYTE
	}
	cache, cacheErr := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     memSize,
		BufferItems: 64,
	})
	if cacheErr != nil {
		err = errors.Warning("create local cache failed").WithCause(cacheErr)
		return
	}
	store = &localStore{
		lock:   sync.RWMutex{},
		cache:  cache,
		values: smap.New(),
	}
	return
}

type localStore struct {
	lock   sync.RWMutex
	cache  *ristretto.Cache
	values *smap.Map
}

func (store *localStore) Set(ctx context.Context, key []byte, value []byte) (err errors.CodeError) {
	store.lock.Lock()
	store.lock.Unlock()
	store.cache.Set(key, value, int64(len(value)))
	store.values.Set(mmhash.MemHash(key), value)
	return
}

func (store *localStore) SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err errors.CodeError) {
	store.lock.Lock()
	store.lock.Unlock()
	store.cache.SetWithTTL(key, value, int64(len(value)), ttl)
	return
}

func (store *localStore) ExpireKey(ctx context.Context, key []byte, ttl time.Duration) (err errors.CodeError) {
	store.lock.Lock()
	store.lock.Unlock()
	v, has := store.cache.Get(key)
	if !has {
		return
	}
	vv := v.([]byte)
	store.cache.SetWithTTL(key, v, int64(len(vv)), ttl)
	return
}

func (store *localStore) Incr(ctx context.Context, key []byte, delta int64) (v int64, err errors.CodeError) {
	store.lock.Lock()
	store.lock.Unlock()
	value, has := store.get(ctx, key)
	if has {
		n, _ := binary.Varint(value)
		v = n + delta
	} else {
		v = delta
	}
	p := make([]byte, 10)
	binary.PutVarint(p, v)
	store.cache.Set(key, p, int64(len(p)))
	store.values.Set(mmhash.MemHash(key), p)
	return
}

func (store *localStore) Get(ctx context.Context, key []byte) (value []byte, has bool, err errors.CodeError) {
	store.lock.RLock()
	defer store.lock.RUnlock()
	value, has = store.get(ctx, key)
	return
}

func (store *localStore) get(ctx context.Context, key []byte) (value []byte, has bool) {
	var v interface{}
	v, has = store.cache.Get(key)
	if !has {
		vv, got := store.values.Get(mmhash.MemHash(key))
		if got {
			v, has = vv.([]byte)
		}
		return
	}
	value, has = v.([]byte)
	if !has {
		_ = store.Remove(ctx, key)
		return
	}
	return
}

func (store *localStore) Remove(ctx context.Context, key []byte) (err errors.CodeError) {
	store.lock.Lock()
	store.lock.Unlock()
	store.cache.Del(key)
	store.values.Delete(mmhash.MemHash(key))
	return
}

func (store *localStore) Close() {
	store.cache.Close()
}

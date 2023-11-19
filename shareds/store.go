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

package shareds

import (
	"encoding/binary"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/caches"
	"github.com/aacfactory/fns/commons/container/smap"
	"github.com/aacfactory/fns/commons/mmhash"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/logs"
	"sync"
	"time"
)

type Store interface {
	Get(ctx context.Context, key []byte) (value []byte, has bool, err error)
	Set(ctx context.Context, key []byte, value []byte) (err error)
	SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error)
	Incr(ctx context.Context, key []byte, delta int64) (v int64, err error)
	Remove(ctx context.Context, key []byte) (err error)
	Close()
}

var (
	localStoreBuilder LocalStoreBuild = defaultLocalStoreBuild
)

func RegisterLocalStoreBuild(build LocalStoreBuild) {
	localStoreBuilder = build
}

type LocalStoreBuild func(log logs.Logger, config configures.Config) (Store, error)

type DefaultLocalSharedStoreConfig struct {
	CacheSize string `json:"cacheSize,omitempty" yaml:"cacheSize,omitempty"`
}

func defaultLocalStoreBuild(log logs.Logger, config configures.Config) (store Store, err error) {
	cfg := DefaultLocalSharedStoreConfig{}
	cfgErr := config.As(&cfg)
	if cfgErr != nil {
		err = errors.Warning("fns: build default local shared store failed").WithCause(cfgErr)
		return
	}
	if cfg.CacheSize == "" {
		cfg.CacheSize = "64MB"
	}
	cacheSize, cacheSizeErr := bytex.ParseBytes(cfg.CacheSize)
	if cacheSizeErr != nil {
		err = errors.Warning("fns: build default local shared store failed").WithCause(errors.Warning("parse cacheSize failed").WithCause(cacheSizeErr))
		return
	}
	cache := caches.New(cacheSize)
	store = &localStore{
		locker:    new(sync.RWMutex),
		cache:     cache,
		persisted: smap.New(),
	}
	return
}

type localStore struct {
	locker    *sync.RWMutex
	cache     *caches.Cache
	persisted *smap.Map
}

func (store *localStore) Get(_ context.Context, key []byte) (value []byte, has bool, err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store get failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.locker.RLock()
	defer store.locker.RUnlock()
	value, has = store.cache.Get(key)
	if has {
		return
	}
	k := mmhash.MemHash(key)
	var p interface{}
	p, has = store.persisted.Get(k)
	if has {
		value, has = p.([]byte)
		return
	}
	return
}

func (store *localStore) Set(_ context.Context, key []byte, value []byte) (err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store set failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.locker.Lock()
	defer store.locker.Unlock()
	k := mmhash.MemHash(key)
	store.persisted.Set(k, value)
	return
}

func (store *localStore) SetWithTTL(_ context.Context, key []byte, value []byte, ttl time.Duration) (err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store set failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.locker.Lock()
	defer store.locker.Unlock()
	err = store.cache.SetWithTTL(key, value, ttl)
	if err != nil {
		err = errors.Warning("fns: shared store set failed").WithCause(err).WithMeta("key", string(key))
		return
	}
	return
}

func (store *localStore) Incr(_ context.Context, key []byte, delta int64) (v int64, err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store incr failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.locker.Lock()
	defer store.locker.Unlock()
	k := mmhash.MemHash(key)
	n := int64(0)
	p, has := store.persisted.Get(k)
	if has {
		encoded, ok := p.([]byte)
		if !ok {
			err = errors.Warning("fns: shared store incr failed").WithCause(fmt.Errorf("value of key is not varint bytes")).WithMeta("key", string(key))
			return
		}
		n, _ = binary.Varint(encoded)
	}
	v = n + delta
	encoded := make([]byte, 10)
	binary.PutVarint(encoded, n)
	store.persisted.Set(k, encoded)
	return
}

func (store *localStore) Remove(_ context.Context, key []byte) (err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store remove failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.locker.Lock()
	defer store.locker.Unlock()
	store.cache.Remove(key)
	k := mmhash.MemHash(key)
	store.persisted.Delete(k)
	return
}

func (store *localStore) Close() {
}

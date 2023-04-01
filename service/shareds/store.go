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
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/caches"
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
	store = &localStore{
		cache: caches.New(uint64(memSize)),
	}
	return
}

type localStore struct {
	cache *caches.Cache
}

func (store *localStore) Set(ctx context.Context, key []byte, value []byte) (err errors.CodeError) {
	setErr := store.cache.Set(key, value)
	if setErr != nil {
		err = errors.Warning("fns: shared store set failed").WithCause(setErr).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	return
}

func (store *localStore) SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err errors.CodeError) {
	setErr := store.cache.SetWithTTL(key, value, ttl)
	if setErr != nil {
		err = errors.Warning("fns: shared store set failed").WithCause(setErr).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	return
}

func (store *localStore) ExpireKey(ctx context.Context, key []byte, ttl time.Duration) (err errors.CodeError) {
	setErr := store.cache.Expire(key, ttl)
	if setErr != nil {
		err = errors.Warning("fns: shared store expire key failed").WithCause(setErr).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	return
}

func (store *localStore) Incr(ctx context.Context, key []byte, delta int64) (v int64, err errors.CodeError) {
	n, setErr := store.cache.Incr(key, delta)
	if setErr != nil {
		err = errors.Warning("fns: shared incr failed").WithCause(setErr).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	v = n
	return
}

func (store *localStore) Get(ctx context.Context, key []byte) (value []byte, has bool, err errors.CodeError) {
	value, has = store.cache.Get(key)
	return
}

func (store *localStore) Remove(ctx context.Context, key []byte) (err errors.CodeError) {
	store.cache.Remove(key)
	return
}

func (store *localStore) Close() {
}

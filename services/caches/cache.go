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

package caches

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/logs"
	"time"
)

type KeyParam interface {
	CacheKey(ctx context.Context) (key []byte, err error)
}

type Store interface {
	services.Component
	Get(ctx context.Context, key []byte) (value []byte, has bool, err error)
	Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error)
	Remove(ctx context.Context, key []byte) (err error)
}

type defaultStore struct {
	log    logs.Logger
	prefix []byte
}

func (store *defaultStore) Name() (name string) {
	return "default"
}

func (store *defaultStore) Construct(options services.Options) (err error) {
	store.log = options.Log
	store.prefix = []byte("fns:caches:")
	return
}

func (store *defaultStore) Shutdown(_ context.Context) {
	return
}

func (store *defaultStore) Get(ctx context.Context, key []byte) (value []byte, has bool, err error) {
	st := runtime.SharedStore(ctx)
	value, has, err = st.Get(ctx, append(store.prefix, key...))
	if err != nil {
		err = errors.Warning("fns: get cache failed").WithMeta("key", string(key)).WithCause(err)
		return
	}
	return
}

func (store *defaultStore) Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error) {
	st := runtime.SharedStore(ctx)
	err = st.SetWithTTL(ctx, append(store.prefix, key...), value, ttl)
	if err != nil {
		err = errors.Warning("fns: set cache failed").WithMeta("key", string(key)).WithCause(err)
		return
	}
	return
}

func (store *defaultStore) Remove(ctx context.Context, key []byte) (err error) {
	st := runtime.SharedStore(ctx)
	err = st.Remove(ctx, append(store.prefix, key...))
	if err != nil {
		err = errors.Warning("fns: remove cache failed").WithMeta("key", string(key)).WithCause(err)
		return
	}
	return
}

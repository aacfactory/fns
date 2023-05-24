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
	"bytes"
	"context"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/caches"
	"time"
)

type Caches interface {
	Get(ctx context.Context, key []byte, options ...Option) (value []byte, has bool)
	Exist(ctx context.Context, key []byte, options ...Option) (has bool)
	Set(ctx context.Context, key []byte, value []byte, ttl time.Duration, options ...Option) (prev []byte, ok bool)
	Remove(ctx context.Context, key []byte, options ...Option)
}

func LocalCaches(maxCacheSize uint64) Caches {
	return &localCaches{
		store: caches.New(maxCacheSize),
	}
}

type localCaches struct {
	store *caches.Cache
}

func (cache *localCaches) Get(ctx context.Context, key []byte, options ...Option) (value []byte, has bool) {
	opt, optErr := NewOptions(options)
	if optErr != nil {
		return
	}
	key = bytes.Join([][]byte{bytex.FromString(opt.Scope), key}, []byte{'/'})
	value, has = cache.store.Get(key)
	return
}

func (cache *localCaches) Exist(ctx context.Context, key []byte, options ...Option) (has bool) {
	opt, optErr := NewOptions(options)
	if optErr != nil {
		return
	}
	key = bytes.Join([][]byte{bytex.FromString(opt.Scope), key}, []byte{'/'})
	has = cache.store.Exist(key)
	return
}

func (cache *localCaches) Set(ctx context.Context, key []byte, value []byte, ttl time.Duration, options ...Option) (prev []byte, ok bool) {
	opt, optErr := NewOptions(options)
	if optErr != nil {
		return
	}
	key = bytes.Join([][]byte{bytex.FromString(opt.Scope), key}, []byte{'/'})
	old, has := cache.store.Get(key)
	if has {
		prev = old
	}
	ok = cache.store.SetWithTTL(key, value, ttl) == nil
	return
}

func (cache *localCaches) Remove(ctx context.Context, key []byte, options ...Option) {
	opt, optErr := NewOptions(options)
	if optErr != nil {
		return
	}
	key = bytes.Join([][]byte{bytex.FromString(opt.Scope), key}, []byte{'/'})
	cache.store.Remove(key)
	return
}

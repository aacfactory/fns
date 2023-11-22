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

package cachecontrol

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key []byte) (value []byte, has bool, err error)
	Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error)
	Close()
}

var (
	cacheKeyPrefix = []byte("fns:cachecontrol:")
)

type DefaultCache struct{}

func (cache *DefaultCache) Get(ctx context.Context, key []byte) (value []byte, has bool, err error) {
	store := runtime.SharedStore(ctx)
	value, has, err = store.Get(ctx, append(cacheKeyPrefix, key...))
	if err != nil {
		err = errors.Warning("fns: cache control store get failed").WithCause(err)
		return
	}
	return
}

func (cache *DefaultCache) Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error) {
	store := runtime.SharedStore(ctx)
	err = store.SetWithTTL(ctx, append(cacheKeyPrefix, key...), value, ttl)
	if err != nil {
		err = errors.Warning("fns: cache control store set failed").WithCause(err)
		return
	}
	return
}

func (cache *DefaultCache) Close() {
}

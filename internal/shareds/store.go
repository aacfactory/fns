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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/dgraph-io/ristretto"
	"time"
)

func NewLocalStore(memSize int64) (store *LocalStore, err error) {
	if memSize < 1 {
		memSize = 64 * commons.MEGABYTE
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
	store = &LocalStore{
		cache: cache,
	}
	return
}

type LocalStore struct {
	cache *ristretto.Cache
}

func (store *LocalStore) Set(key []byte, value []byte, timeout time.Duration) (err errors.CodeError) {
	store.cache.SetWithTTL(key, value, int64(len(value)), timeout)
	return
}

func (store *LocalStore) Get(key []byte) (value []byte, err errors.CodeError) {
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

func (store *LocalStore) Remove(key []byte) (err errors.CodeError) {
	store.cache.Del(key)
	return
}

func (store *LocalStore) Close() {
	store.cache.Close()
}

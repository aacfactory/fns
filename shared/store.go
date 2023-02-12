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
	"fmt"
	"github.com/aacfactory/errors"
	"time"
)

const (
	storeContextKey = "@fns_shared_store"
)

func WithStore(ctx context.Context, store Store) context.Context {
	return context.WithValue(ctx, storeContextKey, store)
}

type Store interface {
	Set(key []byte, value []byte, timeout time.Duration) (err errors.CodeError)
	Get(key []byte) (value []byte, err errors.CodeError)
	Remove(key []byte) (err errors.CodeError)
	Close()
}

func getStore(ctx context.Context) (store Store, err errors.CodeError) {
	v := ctx.Value(storeContextKey)
	if v == nil {
		err = errors.Warning("fns: get shared store failed").WithCause(fmt.Errorf("not exist"))
		return
	}
	ok := false
	store, ok = v.(Store)
	if !ok {
		err = errors.Warning("fns: get shared store failed").WithCause(fmt.Errorf("type is not matched"))
		return
	}
	return
}

func Get(ctx context.Context, key []byte) (value []byte, err errors.CodeError) {
	store, getStoreErr := getStore(ctx)
	if getStoreErr != nil {
		err = errors.Warning("fns: get from shared store failed").WithCause(getStoreErr)
		return
	}
	value, err = store.Get(key)
	if err != nil {
		err = errors.Warning("fns: get from shared store failed").WithCause(err)
		return
	}
	return
}

func Set(ctx context.Context, key []byte, value []byte, timeout time.Duration) (err errors.CodeError) {
	store, getStoreErr := getStore(ctx)
	if getStoreErr != nil {
		err = errors.Warning("fns: set into shared store failed").WithCause(getStoreErr)
		return
	}
	err = store.Set(key, value, timeout)
	if err != nil {
		err = errors.Warning("fns: set int shared store failed").WithCause(err)
		return
	}
	return
}

func Remove(ctx context.Context, key []byte) (err errors.CodeError) {
	store, getStoreErr := getStore(ctx)
	if getStoreErr != nil {
		err = errors.Warning("fns: remove from shared store failed").WithCause(getStoreErr)
		return
	}
	err = store.Remove(key)
	if err != nil {
		err = errors.Warning("fns: remove from shared store failed").WithCause(err)
		return
	}
	return
}

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
	"encoding/binary"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
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

func LocalStore() (store Store) {
	store = &localStore{
		values: sync.Map{},
	}
	return
}

type entry struct {
	lock     *sync.Mutex
	value    []byte
	deadline time.Time
}

type localStore struct {
	values sync.Map
}

func (store *localStore) Set(ctx context.Context, key []byte, value []byte) (err errors.CodeError) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store set failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.values.Store(bytex.ToString(key), &entry{
		lock:     new(sync.Mutex),
		value:    value,
		deadline: time.Time{},
	})
	return
}

func (store *localStore) SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err errors.CodeError) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store set failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.values.Store(bytex.ToString(key), &entry{
		value:    value,
		deadline: time.Now().Add(ttl),
	})
	return
}

func (store *localStore) ExpireKey(ctx context.Context, key []byte, ttl time.Duration) (err errors.CodeError) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store expire key failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	k := bytex.ToString(key)
	x, loaded := store.values.Load(k)
	if !loaded {
		err = errors.Warning("fns: shared store expire key failed").WithCause(errors.Warning("key was not found")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	e := x.(*entry)
	e.deadline = time.Now().Add(ttl)
	store.values.Store(k, e)
	return
}

func (store *localStore) Incr(ctx context.Context, key []byte, delta int64) (v int64, err errors.CodeError) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store incr failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	k := bytex.ToString(key)
	x, _ := store.values.LoadOrStore(k, &entry{value: make([]byte, 10)})
	e := x.(*entry)
	e.lock.Lock()
	n, _ := binary.Varint(e.value)
	if !e.deadline.IsZero() && e.deadline.Before(time.Now()) {
		n = 0
	}
	n += delta
	binary.PutVarint(e.value, n)
	e.lock.Unlock()
	v = n
	return
}

func (store *localStore) Get(ctx context.Context, key []byte) (value []byte, has bool, err errors.CodeError) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store get failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	k := bytex.ToString(key)
	x, loaded := store.values.Load(k)
	if !loaded {
		return
	}
	e := x.(*entry)
	if !e.deadline.IsZero() && e.deadline.Before(time.Now()) {
		store.values.Delete(k)
		return
	}
	value = e.value
	has = true
	return
}

func (store *localStore) Remove(ctx context.Context, key []byte) (err errors.CodeError) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store remove failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	k := bytex.ToString(key)
	store.values.Delete(k)
	return
}

func (store *localStore) Close() {
}

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

package shareds

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/logs"
	"github.com/tidwall/btree"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Store interface {
	Get(ctx context.Context, key []byte) (value []byte, has bool, err error)
	Set(ctx context.Context, key []byte, value []byte) (err error)
	SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error)
	Remove(ctx context.Context, key []byte) (err error)
	Incr(ctx context.Context, key []byte, delta int64) (v int64, err error)
	Expire(ctx context.Context, key []byte, ttl time.Duration) (err error)
	Close()
}

type StoreBuilder interface {
	Build(ctx context.Context, config configures.Config) (store Store, err error)
}

var (
	localStoreBuilder LocalStoreBuild = defaultLocalStoreBuild
)

func RegisterLocalStoreBuild(build LocalStoreBuild) {
	localStoreBuilder = build
}

type LocalStoreBuild func(log logs.Logger, config configures.Config) (Store, error)

type DefaultLocalSharedStoreConfig struct {
	ShrinkDuration time.Duration `json:"shrinkDuration"`
}

func defaultLocalStoreBuild(log logs.Logger, config configures.Config) (store Store, err error) {
	cfg := DefaultLocalSharedStoreConfig{}
	cfgErr := config.As(&cfg)
	if cfgErr != nil {
		err = errors.Warning("fns: build default local shared store failed").WithCause(cfgErr)
		return
	}
	shrinkDuration := cfg.ShrinkDuration
	if shrinkDuration == 0 {
		shrinkDuration = 1 * time.Hour
	}
	s := &localStore{
		log:            log,
		shrinkDuration: shrinkDuration,
		closeCh:        make(chan struct{}, 1),
		locker:         sync.RWMutex{},
		values:         btree.NewMap[string, Entry](0),
	}
	if shrinkDuration > 0 {
		go s.shrink()
	}
	store = s
	return
}

type localStore struct {
	log            logs.Logger
	shrinkDuration time.Duration
	closeCh        chan struct{}
	locker         sync.RWMutex
	values         *btree.Map[string, Entry]
}

func (store *localStore) Get(_ context.Context, key []byte) (value []byte, has bool, err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store get failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.locker.RLock()
	defer store.locker.RUnlock()
	sk := bytex.ToString(key)
	entry, exist := store.values.Get(sk)
	if !exist {
		return
	}
	if entry.Expired() {
		return
	}
	has = true
	switch v := entry.Value.(type) {
	case []byte:
		value = v
		break
	case *atomic.Int64:
		value = bytex.FromString(strconv.FormatInt(v.Load(), 64))
		break
	default:
		has = false
		break
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
	sk := bytex.ToString(key)
	store.values.Set(sk, Entry{
		key:      sk,
		Value:    value,
		Deadline: time.Time{},
	})
	return
}

func (store *localStore) SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store set failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	if ttl < 1 {
		err = store.Set(ctx, key, value)
		return
	}
	store.locker.Lock()
	defer store.locker.Unlock()
	sk := bytex.ToString(key)
	store.values.Set(sk, Entry{
		key:      sk,
		Value:    value,
		Deadline: time.Now().Add(ttl),
	})
	return
}

func (store *localStore) Incr(_ context.Context, key []byte, delta int64) (v int64, err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store incr failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.locker.Lock()
	defer store.locker.Unlock()
	sk := bytex.ToString(key)
	var n *atomic.Int64
	entry, exist := store.values.Get(sk)
	if exist {
		if entry.Expired() {
			n = new(atomic.Int64)
		} else {
			nn, ok := entry.Value.(*atomic.Int64)
			if !ok {
				err = errors.Warning("fns: shared store incr failed").WithCause(errors.Warning("value of key is not int64")).WithMeta("shared", "local").WithMeta("key", string(key))
				return
			}
			n = nn
		}
	} else {
		n = new(atomic.Int64)
		entry = Entry{
			key:      sk,
			Value:    n,
			Deadline: time.Time{},
		}
	}
	v = n.Add(delta)
	store.values.Set(sk, entry)
	return
}

func (store *localStore) Remove(_ context.Context, key []byte) (err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store remove failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.locker.Lock()
	defer store.locker.Unlock()
	sk := bytex.ToString(key)
	store.values.Delete(sk)
	return
}

func (store *localStore) Expire(ctx context.Context, key []byte, ttl time.Duration) (err error) {
	if key == nil || len(key) == 0 {
		err = errors.Warning("fns: shared store expire key failed").WithCause(errors.Warning("key is required")).WithMeta("shared", "local").WithMeta("key", string(key))
		return
	}
	store.locker.Lock()
	defer store.locker.Unlock()
	sk := bytex.ToString(key)
	entry, exist := store.values.Get(sk)
	if !exist {
		return
	}
	if ttl < 1 {
		entry.Deadline = time.Time{}
	} else {
		entry.Deadline = time.Now().Add(ttl)
	}
	store.values.Set(sk, entry)
	return
}

func (store *localStore) shrink() {
	timer := time.NewTimer(store.shrinkDuration)
	stop := false
	for {
		select {
		case <-store.closeCh:
			stop = true
			break
		case <-timer.C:
			store.locker.Lock()
			expires := make([]string, 0, 1)
			values := store.values.Values()
			for _, value := range values {
				if value.Expired() {
					expires = append(expires, value.key)
				}
			}
			for _, expire := range expires {
				store.values.Delete(expire)
			}
			store.locker.Unlock()
			timer.Reset(store.shrinkDuration)
			break
		}
		if stop {
			break
		}
	}
	timer.Stop()
}

func (store *localStore) Close() {
	if store.shrinkDuration > 0 {
		store.closeCh <- struct{}{}
	}
	close(store.closeCh)
}

type Entry struct {
	key      string
	Value    any
	Deadline time.Time
}

func (entry *Entry) Expired() bool {
	if entry.Deadline.IsZero() {
		return false
	}
	return time.Now().After(entry.Deadline)
}

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

package authorizations

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/avro"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"time"
)

type TokenStore interface {
	services.Component
	Get(ctx context.Context, account Id, id Id) (v Authorization, has bool, err error)
	List(ctx context.Context, account Id) (v []Authorization, err error)
	Save(ctx context.Context, v Authorization) (err error)
	Remove(ctx context.Context, account Id, ids []Id) (err error)
}

func SharingTokenStore() TokenStore {
	return &sharingTokenStore{}
}

type sharingTokenStore struct {
	keyPrefix []byte
}

func (store *sharingTokenStore) Name() (name string) {
	return "store"
}

func (store *sharingTokenStore) Construct(_ services.Options) (err error) {
	store.keyPrefix = []byte("fns:authorizations:")
	return
}

func (store *sharingTokenStore) Shutdown(_ context.Context) {
	return
}

func (store *sharingTokenStore) Get(ctx context.Context, account Id, id Id) (v Authorization, has bool, err error) {
	if !id.Exist() {
		err = errors.Warning("authorizations: get authorization failed").WithCause(fmt.Errorf("id is required"))
		return
	}
	entries, listErr := store.List(ctx, account)
	if listErr != nil {
		err = errors.Warning("authorizations: get authorization failed").WithCause(listErr)
		return
	}
	for _, entry := range entries {
		if bytes.Equal(entry.Id, id) {
			v = entry
			has = true
			return
		}
	}
	return
}

func (store *sharingTokenStore) List(ctx context.Context, account Id) (v []Authorization, err error) {
	if !account.Exist() {
		err = errors.Warning("authorizations: list authorization failed").WithCause(fmt.Errorf("account is required"))
		return
	}
	sc := runtime.SharedStore(ctx)
	key := append(store.keyPrefix, account...)
	p, has, getErr := sc.Get(ctx, key)
	if getErr != nil {
		err = errors.Warning("authorizations: list authorization failed").WithCause(getErr)
		return
	}
	if !has {
		return
	}
	v = make([]Authorization, 0)
	err = avro.Unmarshal(p, &v)
	if err != nil {
		err = errors.Warning("authorizations: list authorization failed").WithCause(err)
		return
	}
	return
}

func (store *sharingTokenStore) Save(ctx context.Context, v Authorization) (err error) {
	if !v.Exist() || !v.Validate() {
		return
	}
	entries, listErr := store.List(ctx, v.Account)
	if listErr != nil {
		err = errors.Warning("authorizations: save authorization failed").WithCause(listErr)
		return
	}
	entries = append(entries, v)
	expire := time.Time{}
	for _, entry := range entries {
		if entry.ExpireAT.After(expire) {
			expire = entry.ExpireAT
		}
	}
	p, encodeErr := avro.Marshal(entries)
	if encodeErr != nil {
		err = errors.Warning("authorizations: save authorization failed").WithCause(encodeErr)
		return
	}
	sc := runtime.SharedStore(ctx)
	key := append(store.keyPrefix, v.Account...)
	setErr := sc.SetWithTTL(ctx, key, p, expire.Sub(time.Now()))
	if setErr != nil {
		err = errors.Warning("authorizations: save authorization failed").WithCause(setErr)
		return
	}
	return
}

func (store *sharingTokenStore) Remove(ctx context.Context, account Id, ids []Id) (err error) {
	entries, listErr := store.List(ctx, account)
	if listErr != nil {
		err = errors.Warning("authorizations: remove authorization failed").WithCause(listErr)
		return
	}
	if len(entries) == 0 {
		return
	}
	sc := runtime.SharedStore(ctx)
	key := append(store.keyPrefix, account...)
	if len(ids) == 0 {
		rmErr := sc.Remove(ctx, key)
		if rmErr != nil {
			err = errors.Warning("authorizations: remove authorization failed").WithCause(rmErr)
			return
		}
		return
	}
	news := make([]Authorization, 0, 1)
	expire := time.Time{}
	for _, entry := range entries {
		if !entry.Validate() {
			continue
		}
		in := false
		for _, id := range ids {
			if bytes.Equal(id, entry.Id) {
				in = true
				break
			}
		}
		if in {
			continue
		}
		news = append(news, entry)
		if entry.ExpireAT.After(expire) {
			expire = entry.ExpireAT
		}
	}
	if len(news) == 0 {
		rmErr := sc.Remove(ctx, key)
		if rmErr != nil {
			err = errors.Warning("authorizations: remove authorization failed").WithCause(rmErr)
			return
		}
		return
	}
	p, encodeErr := avro.Marshal(news)
	if encodeErr != nil {
		err = errors.Warning("authorizations: remove authorization failed").WithCause(encodeErr)
		return
	}
	setErr := sc.SetWithTTL(ctx, key, p, expire.Sub(time.Now()))
	if setErr != nil {
		err = errors.Warning("authorizations: remove authorization failed").WithCause(setErr)
		return
	}
	return
}

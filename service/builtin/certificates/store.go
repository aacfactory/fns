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

package certificates

import (
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"time"
)

type Certificate struct {
	Id          string    `json:"id"`
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Key         []byte    `json:"key"`
	SecretKey   []byte    `json:"secretKey"`
	Password    []byte    `json:"password"`
	ExpireAT    time.Time `json:"expireAT"`
}

type StoreOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	Log        logs.Logger
	Config     configures.Config
}

type Store interface {
	Build(options StoreOptions) (err error)
	Get(ctx context.Context, id string) (certificate *Certificate, err errors.CodeError)
	Create(ctx context.Context, certificate *Certificate) (err errors.CodeError)
	Remove(ctx context.Context, id string) (err errors.CodeError)
	Close()
}

func convertStoreToComponent(store Store) service.Component {
	return &storeComponent{
		store: store,
	}
}

const (
	storeComponentName = "store"
)

type storeComponent struct {
	store Store
}

func (component *storeComponent) Name() string {
	return storeComponentName
}

func (component *storeComponent) Build(options service.ComponentOptions) (err error) {
	err = component.store.Build(StoreOptions{
		AppId:      options.AppId,
		AppName:    options.AppName,
		AppVersion: options.AppVersion,
		Log:        options.Log,
		Config:     options.Config,
	})
	return
}

func (component *storeComponent) Close() {
	component.store.Close()
}

func DefaultCertificates() Store {
	return &defaultCertificates{}
}

type defaultCertificates struct {
}

func (certs *defaultCertificates) Build(_ StoreOptions) (err error) {
	return
}

func (certs *defaultCertificates) key(id string) string {
	return fmt.Sprintf("fns/certificates/%s", id)
}

func (certs *defaultCertificates) Get(ctx context.Context, id string) (certificate *Certificate, err errors.CodeError) {
	if id == "" {
		err = errors.Warning("certificates: get certificate failed").WithCause(errors.Warning("id is required"))
		return
	}
	// todo cache 移到外面去，然后是local的，
	key := bytex.FromString(certs.key(id))
	cache := service.SharedCache(ctx)
	v, has := cache.Get(ctx, key)
	if !has {
		store := service.SharedStore(ctx)
		v, has, err = store.Get(ctx, bytex.FromString(certs.key(id)))
		if err != nil {
			err = errors.Warning("certificates: get certificate failed").WithCause(err)
			return
		}
		if !has {
			return
		}
		_, _ = cache.Set(ctx, key, v, 24*time.Hour)
	}
	certificate = &Certificate{}
	decodeErr := json.Unmarshal(v, certificate)
	if decodeErr != nil {
		err = errors.Warning("certificates: get certificate failed").WithCause(decodeErr)
		return
	}
	return
}

func (certs *defaultCertificates) Create(ctx context.Context, certificate *Certificate) (err errors.CodeError) {
	if certificate == nil {
		err = errors.Warning("certificates: create certificate failed").WithCause(errors.Warning("certificate is required"))
		return
	}
	id := certificate.Id
	if id == "" {
		err = errors.Warning("certificates: create certificate failed").WithCause(errors.Warning("id is required"))
		return
	}
	p, encodeErr := json.Marshal(certificate)
	if encodeErr != nil {
		err = errors.Warning("certificates: create certificate failed").WithCause(encodeErr)
		return
	}
	store := service.SharedStore(ctx)
	setErr := store.Set(ctx, bytex.FromString(certs.key(id)), p)
	if setErr != nil {
		err = errors.Warning("certificates: create certificate failed").WithCause(setErr)
		return
	}
	return
}

func (certs *defaultCertificates) Remove(ctx context.Context, id string) (err errors.CodeError) {
	if id == "" {
		err = errors.Warning("certificates: remove certificate failed").WithCause(errors.Warning("id is required"))
		return
	}
	key := bytex.FromString(certs.key(id))
	cache := service.SharedCache(ctx)
	cache.Remove(ctx, key)
	store := service.SharedStore(ctx)
	rmErr := store.Remove(ctx, key)
	if rmErr != nil {
		err = errors.Warning("certificates: remove certificate failed").WithCause(rmErr)
		return
	}
	return
}

func (certs *defaultCertificates) Close() {
	return
}

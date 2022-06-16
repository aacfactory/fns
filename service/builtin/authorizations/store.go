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

package authorizations

import (
	"context"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/logs"
)

type TokenStoreOptions struct {
	Log    logs.Logger
	Config configuares.Config
}

type TokenStore interface {
	Build(options TokenStoreOptions) (err error)
	Exist(ctx context.Context, tokenId string) (ok bool)
	Save(ctx context.Context, token Token) (err error)
	Remove(ctx context.Context, tokenId string) (err error)
}

type tokenStoreComponent struct {
	store TokenStore
}

func (component *tokenStoreComponent) Name() (name string) {
	name = "store"
	return
}

func (component *tokenStoreComponent) Build(options service.ComponentOptions) (err error) {
	config, hasConfig := options.Config.Node("store")
	if !hasConfig {
		err = errors.Warning("fns: build authorizations token store failed").WithCause(fmt.Errorf("there is no store node in authorizations config node"))
		return
	}
	err = component.store.Build(TokenStoreOptions{
		Log:    options.Log,
		Config: config,
	})
	return
}

func (component *tokenStoreComponent) Exist(ctx context.Context, tokenId string) (ok bool) {
	ok = component.store.Exist(ctx, tokenId)
	return
}

func (component *tokenStoreComponent) Save(ctx context.Context, token Token) (err error) {
	err = component.store.Save(ctx, token)
	return
}

func (component *tokenStoreComponent) Remove(ctx context.Context, tokenId string) (err error) {
	err = component.store.Remove(ctx, tokenId)
	return
}

func (component *tokenStoreComponent) Close() {
}

func DiscardTokenStore() TokenStore {
	return &discardTokenStore{}
}

type discardTokenStore struct {
}

func (store *discardTokenStore) Build(options TokenStoreOptions) (err error) {
	return
}

func (store *discardTokenStore) Exist(ctx context.Context, tokenId string) (ok bool) {
	ok = true
	return
}

func (store *discardTokenStore) Save(ctx context.Context, token Token) (err error) {
	return
}

func (store *discardTokenStore) Remove(ctx context.Context, tokenId string) (err error) {
	return
}

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

import "github.com/aacfactory/fns"

type TokenStore interface {
	Build(env fns.Environments) (err error)
	Exist(ctx fns.Context, tokenId string) (ok bool)
	Save(ctx fns.Context, token Token) (err error)
	Remove(ctx fns.Context, token Token) (err error)
}

type tokenStoreComponent struct {
	store TokenStore
}

func (store *tokenStoreComponent) Name() (name string) {
	name = "store"
	return
}

func (store *tokenStoreComponent) Build(env fns.Environments) (err error) {
	err = store.store.Build(env)
	return
}

func (store *tokenStoreComponent) Exist(ctx fns.Context, tokenId string) (ok bool) {
	ok = store.store.Exist(ctx, tokenId)
	return
}

func (store *tokenStoreComponent) Save(ctx fns.Context, token Token) (err error) {
	err = store.store.Save(ctx, token)
	return
}

func (store *tokenStoreComponent) Remove(ctx fns.Context, token Token) (err error) {
	err = store.store.Remove(ctx, token)
	return
}

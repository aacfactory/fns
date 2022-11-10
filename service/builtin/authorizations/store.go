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
	"github.com/aacfactory/fns/service"
)

type TokenStoreComponent interface {
	service.Component
	Exist(ctx context.Context, tokenId string) (ok bool)
	Save(ctx context.Context, token Token) (err error)
	Remove(ctx context.Context, tokenId string) (err error)
	RemoveUserTokens(ctx context.Context, userId string) (err error)
}

func DefaultTokenStoreComponent() TokenStoreComponent {
	return &defaultTokenStoreComponent{}
}

type defaultTokenStoreComponent struct {
}

func (component *defaultTokenStoreComponent) Name() (name string) {
	name = "store"
	return
}

func (component *defaultTokenStoreComponent) Build(options service.ComponentOptions) (err error) {
	return
}

func (component *defaultTokenStoreComponent) Close() {
	return
}

func (component *defaultTokenStoreComponent) Exist(ctx context.Context, tokenId string) (ok bool) {
	ok = true
	return
}

func (component *defaultTokenStoreComponent) Save(ctx context.Context, token Token) (err error) {
	return
}

func (component *defaultTokenStoreComponent) Remove(ctx context.Context, tokenId string) (err error) {
	return
}

func (component *defaultTokenStoreComponent) RemoveUserTokens(ctx context.Context, userId string) (err error) {
	return
}

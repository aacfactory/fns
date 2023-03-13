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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"time"
)

type TokenStoreComponent interface {
	service.Component
	Exist(ctx context.Context, tokenId string) (ok bool, err error)
	Save(ctx context.Context, token Token) (err error)
	Remove(ctx context.Context, userId string, tokenId string) (err error)
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

func (component *defaultTokenStoreComponent) Exist(ctx context.Context, tokenId string) (ok bool, err error) {
	store := service.SharedStore(ctx)
	_, ok, err = store.Get(ctx, bytex.FromString(fmt.Sprintf("authorizations/tokens/%s", tokenId)))
	if err != nil {
		err = errors.Warning("authorizations: check token exist failed").WithCause(err)
		return
	}
	return
}

func (component *defaultTokenStoreComponent) Save(ctx context.Context, token Token) (err error) {
	store := service.SharedStore(ctx)
	userId, _ := token.User()
	if userId == "" {
		err = errors.Warning("authorizations: save token failed").WithCause(errors.Warning("user id is required"))
		return
	}
	tokenBytes, has, getUserTokensErr := store.Get(ctx, bytex.FromString(fmt.Sprintf("authorizations/users/%s", userId)))
	if getUserTokensErr != nil {
		err = errors.Warning("authorizations: save token failed").WithCause(getUserTokensErr)
		return
	}
	if !has {
		tokenBytes = []byte{'[', ']'}
	}
	tokenIds := make([]string, 0, 1)
	decodeErr := json.Unmarshal(tokenBytes, &tokenIds)
	if decodeErr != nil {
		err = errors.Warning("authorizations: save token failed").WithCause(decodeErr)
		return
	}
	tokenIds = append(tokenIds, token.Id())
	tokenBytes, err = json.Marshal(tokenIds)
	if err != nil {
		err = errors.Warning("authorizations: save token failed").WithCause(err)
		return
	}

	ttl := token.NotAfter().Sub(time.Now())
	setTokenErr := store.SetWithTTL(ctx, bytex.FromString(fmt.Sprintf("authorizations/tokens/%s/%s", userId, token.Id())), token.Bytes(), ttl)
	if setTokenErr != nil {
		err = errors.Warning("authorizations: save token failed").WithCause(setTokenErr)
		return
	}
	setUserTokensErr := store.SetWithTTL(ctx, bytex.FromString(fmt.Sprintf("authorizations/users/%s", userId)), tokenBytes, ttl)
	if setUserTokensErr != nil {
		err = errors.Warning("authorizations: save token failed").WithCause(setUserTokensErr)
		return
	}
	return
}

func (component *defaultTokenStoreComponent) Remove(ctx context.Context, userId string, tokenId string) (err error) {
	store := service.SharedStore(ctx)
	rmErr := store.Remove(ctx, bytex.FromString(fmt.Sprintf("authorizations/tokens/%s/%s", userId, tokenId)))
	if rmErr != nil {
		err = errors.Warning("authorizations: remove token failed").WithCause(rmErr)
		return
	}
	tokenBytes, has, getUserTokensErr := store.Get(ctx, bytex.FromString(fmt.Sprintf("authorizations/users/%s", userId)))
	if getUserTokensErr != nil {
		err = errors.Warning("authorizations: remove token failed").WithCause(getUserTokensErr)
		return
	}
	if !has {
		return
	}
	tokenIds := make([]string, 0, 1)
	decodeErr := json.Unmarshal(tokenBytes, &tokenIds)
	if decodeErr != nil {
		err = errors.Warning("authorizations: remove token failed").WithCause(decodeErr)
		return
	}
	newTokenIds := make([]string, 0, 1)
	for _, id := range tokenIds {
		if id == tokenId {
			continue
		}
		newTokenIds = append(newTokenIds, id)
	}
	tokenBytes, err = json.Marshal(newTokenIds)
	if err != nil {
		err = errors.Warning("authorizations: remove token failed").WithCause(err)
		return
	}
	setUserTokensErr := store.Set(ctx, bytex.FromString(fmt.Sprintf("authorizations/users/%s", userId)), tokenBytes)
	if setUserTokensErr != nil {
		err = errors.Warning("authorizations: remove token failed").WithCause(setUserTokensErr)
		return
	}
	return
}

func (component *defaultTokenStoreComponent) RemoveUserTokens(ctx context.Context, userId string) (err error) {
	store := service.SharedStore(ctx)
	tokenBytes, has, getUserTokensErr := store.Get(ctx, bytex.FromString(fmt.Sprintf("authorizations/users/%s", userId)))
	if getUserTokensErr != nil {
		err = errors.Warning("authorizations: remove user tokens failed").WithCause(getUserTokensErr)
		return
	}
	if !has {
		return
	}
	tokenIds := make([]string, 0, 1)
	decodeErr := json.Unmarshal(tokenBytes, &tokenIds)
	if decodeErr != nil {
		err = errors.Warning("authorizations: remove user tokens failed").WithCause(decodeErr)
		return
	}
	for _, id := range tokenIds {
		rmErr := store.Remove(ctx, bytex.FromString(fmt.Sprintf("authorizations/tokens/%s/%s", userId, id)))
		if rmErr != nil {
			err = errors.Warning("authorizations: remove user tokens failed").WithCause(rmErr)
			return
		}
	}
	rmErr := store.Remove(ctx, bytex.FromString(fmt.Sprintf("authorizations/users/%s", userId)))
	if rmErr != nil {
		err = errors.Warning("authorizations: remove user tokens failed").WithCause(rmErr)
		return
	}
	return
}

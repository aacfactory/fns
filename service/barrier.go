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

package service

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service/shared"
	"golang.org/x/sync/singleflight"
)

type Barrier interface {
	Do(ctx context.Context, key string, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError)
	Forget(ctx context.Context, key string)
}

func defaultBarrier() Barrier {
	return &sfgBarrier{
		v: &singleflight.Group{},
	}
}

type sfgBarrier struct {
	v *singleflight.Group
}

func (b *sfgBarrier) Do(_ context.Context, key string, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError) {
	var doErr error
	result, doErr, _ = b.v.Do(key, func() (interface{}, error) {
		return fn()
	})
	if doErr != nil {
		err = doErr.(errors.CodeError)
	}
	return
}

func (b *sfgBarrier) Forget(_ context.Context, key string) {
	b.v.Forget(key)
}

func clusterBarrier(store shared.Store) Barrier {
	return &sharedBarrier{
		store: store,
	}
}

type sharedBarrier struct {
	store shared.Store
}

func (barrier *sharedBarrier) Do(ctx context.Context, key string, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError) {
	//TODO implement me
	panic("implement me")
}

func (barrier *sharedBarrier) Forget(ctx context.Context, key string) {
	//TODO implement me
	panic("implement me")
}

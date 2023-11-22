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

package barriers

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/context"
	"golang.org/x/sync/singleflight"
)

type Result interface {
	scanner.Scanner
}

type Barrier interface {
	Do(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (result Result, err error)
	Forget(ctx context.Context, key []byte)
}

func New() (b Barrier) {
	b = &barrier{
		group: new(singleflight.Group),
	}
	return
}

// Barrier
// @barrier
// 当@authorization 存在时，则key为 services.HashRequest(r, services.HashRequestWithToken())
type barrier struct {
	group *singleflight.Group
}

func (b *barrier) Do(_ context.Context, key []byte, fn func() (result interface{}, err error)) (r Result, err error) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	v, doErr, _ := b.group.Do(bytex.ToString(key), func() (v interface{}, err error) {
		v, err = fn()
		return
	})
	if doErr != nil {
		err = errors.Map(doErr)
		return
	}
	r = scanner.New(v)
	return
}

func (b *barrier) Forget(_ context.Context, key []byte) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	b.group.Forget(bytex.ToString(key))
}

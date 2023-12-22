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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
)

var (
	removeFnName = []byte("remove")
)

func Remove(ctx context.Context, account Id, ids ...Id) (err error) {
	rt := runtime.Load(ctx)
	_, err = rt.Endpoints().Request(
		ctx,
		endpointName, removeFnName,
		removeParam{
			Account: account,
			Ids:     ids,
		},
	)
	return
}

type removeParam struct {
	Account Id   `json:"account" avro:"account"`
	Ids     []Id `json:"ids" avro:"ids"`
}

type removeFn struct {
	store TokenStore
}

func (fn *removeFn) Name() string {
	return string(removeFnName)
}

func (fn *removeFn) Internal() bool {
	return true
}

func (fn *removeFn) Readonly() bool {
	return false
}

func (fn *removeFn) Handle(ctx services.Request) (v any, err error) {
	param, parmaErr := services.ValueOfParam[removeParam](ctx.Param())
	if parmaErr != nil {
		err = errors.Warning("authorizations: remove failed").WithCause(parmaErr)
		return
	}
	rmErr := fn.store.Remove(ctx, param.Account, param.Ids)
	if rmErr != nil {
		err = errors.Warning("authorizations: remove failed").WithCause(rmErr)
		return
	}
	return
}

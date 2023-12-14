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
	listFnName = []byte("list")
)

func List(ctx context.Context, account Id) (v []Authorization, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		endpointName, listFnName,
		account,
	)
	if handleErr != nil {
		err = handleErr
		return
	}
	v, err = services.ValueOfResponse[[]Authorization](response)
	if err != nil {
		err = errors.Warning("authorizations: scan encode value failed").WithCause(err)
		return
	}
	return
}

type listFn struct {
	store TokenStore
}

func (fn *listFn) Name() string {
	return string(listFnName)
}

func (fn *listFn) Internal() bool {
	return true
}

func (fn *listFn) Readonly() bool {
	return false
}

func (fn *listFn) Handle(ctx services.Request) (v any, err error) {
	account, parmaErr := services.ValueOfParam[Id](ctx.Param())
	if parmaErr != nil {
		err = errors.Warning("authorizations: list failed").WithCause(parmaErr)
		return
	}
	entries, listErr := fn.store.List(ctx, account)
	if listErr != nil {
		err = errors.Warning("authorizations: list failed").WithCause(listErr)
		return
	}
	v = entries
	return
}

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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
)

var (
	decodeFnName = []byte("decode")
)

func Decode(ctx context.Context, token Token) (authorization Authorization, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		endpointName, decodeFnName,
		token,
	)
	if handleErr != nil {
		err = handleErr
		return
	}
	authorization, err = services.ValueOfResponse[Authorization](response)
	if err != nil {
		err = errors.Warning("authorizations: scan decode value failed").WithCause(err)
		return
	}
	return
}

type decodeFn struct {
	encoder TokenEncoder
	store   TokenStore
}

func (fn *decodeFn) Name() string {
	return string(decodeFnName)
}

func (fn *decodeFn) Internal() bool {
	return true
}

func (fn *decodeFn) Readonly() bool {
	return false
}

func (fn *decodeFn) Handle(r services.Request) (v interface{}, err error) {
	param, paramErr := services.ValueOfParam[Token](r.Param())
	if paramErr != nil {
		err = errors.Warning("authorizations: invalid param")
		return
	}
	authorization, decodeErr := fn.encoder.Decode(r, param)
	if decodeErr != nil {
		err = errors.Warning("authorizations: decode token failed").WithCause(decodeErr)
		return
	}
	stored, has, getErr := fn.store.Get(r, authorization.Account, authorization.Id)
	if getErr != nil {
		err = errors.Warning("authorizations: decode token failed").WithCause(getErr)
		return
	}
	if !has {
		err = errors.Warning("authorizations: decode token failed").WithCause(fmt.Errorf("not exist"))
		return
	}
	v = stored
	return
}

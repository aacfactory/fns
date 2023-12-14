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
	encodeFnName = []byte("encode")
)

func Encode(ctx context.Context, authorization Authorization) (token Token, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		endpointName, encodeFnName,
		authorization,
	)
	if handleErr != nil {
		err = handleErr
		return
	}
	token, err = services.ValueOfResponse[Token](response)
	if err != nil {
		err = errors.Warning("authorizations: scan encode value failed").WithCause(err)
		return
	}
	return
}

type encodeFn struct {
	encoder TokenEncoder
	store   TokenStore
}

func (fn *encodeFn) Name() string {
	return string(encodeFnName)
}

func (fn *encodeFn) Internal() bool {
	return true
}

func (fn *encodeFn) Readonly() bool {
	return false
}

func (fn *encodeFn) Handle(r services.Request) (v interface{}, err error) {
	param, paramErr := services.ValueOfParam[Authorization](r.Param())
	if paramErr != nil {
		err = errors.Warning("authorizations: invalid param")
		return
	}
	token, encodeErr := fn.encoder.Encode(r, param)
	if encodeErr != nil {
		err = errors.Warning("authorizations: encode authorization failed").WithCause(encodeErr)
		return
	}
	saveErr := fn.store.Save(r, param)
	if saveErr != nil {
		err = errors.Warning("authorizations: encode authorization failed").WithCause(saveErr)
		return
	}
	v = token
	return
}

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
	"time"
)

var (
	createFnName = []byte("create")
)

type CreateParam struct {
	Id         Id         `json:"id" avro:"id"`
	Account    Id         `json:"account" avro:"account"`
	Attributes Attributes `json:"attributes" avro:"attributes"`
}

type CreateResult struct {
	Token         Token         `json:"token" avro:"token"`
	Authorization Authorization `json:"authorization" avro:"authorization"`
}

func Create(ctx context.Context, param CreateParam) (token Token, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		endpointName, createFnName,
		param,
	)
	if handleErr != nil {
		err = handleErr
		return
	}
	result, resultErr := services.ValueOfResponse[CreateResult](response)
	if resultErr != nil {
		err = errors.Warning("authorizations: scan encode value failed").WithCause(resultErr)
		return
	}
	With(ctx, result.Authorization)
	token = result.Token
	return
}

type createFn struct {
	encoder   TokenEncoder
	store     TokenStore
	expireTTL time.Duration
}

func (fn *createFn) Name() string {
	return string(createFnName)
}

func (fn *createFn) Internal() bool {
	return true
}

func (fn *createFn) Readonly() bool {
	return false
}

func (fn *createFn) Handle(r services.Request) (v any, err error) {
	param, paramErr := services.ValueOfParam[CreateParam](r.Param())
	if paramErr != nil {
		err = errors.Warning("authorizations: invalid param")
		return
	}
	if len(param.Id) == 0 || len(param.Account) == 0 {
		err = errors.Warning("authorizations: invalid param")
		return
	}
	authorization := Authorization{
		Id:         param.Id,
		Account:    param.Account,
		Attributes: param.Attributes,
		ExpireAT:   time.Now().Add(fn.expireTTL),
	}
	token, encodeErr := fn.encoder.Encode(r, authorization)
	if encodeErr != nil {
		err = errors.Warning("authorizations: encode authorization failed").WithCause(encodeErr)
		return
	}
	saveErr := fn.store.Save(r, authorization)
	if saveErr != nil {
		err = errors.Warning("authorizations: encode authorization failed").WithCause(saveErr)
		return
	}
	v = CreateResult{
		Token:         token,
		Authorization: authorization,
	}
	return
}

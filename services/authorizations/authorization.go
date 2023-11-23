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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"time"
)

var (
	contextUserKey = []byte("authorizations")
)

func With(ctx context.Context, authorization Authorization) {
	ctx.SetUserValue(contextUserKey, authorization)
}

func Load(ctx context.Context) Authorization {
	authorization := Authorization{}
	_, _ = ctx.ScanUserValue(contextUserKey, &authorization)
	return authorization
}

type Authorization struct {
	Id         Id         `json:"id"`
	Account    Id         `json:"account"`
	Attributes Attributes `json:"attributes"`
	ExpireAT   time.Time  `json:"expireAT"`
}

func (authorization Authorization) Exist() bool {
	return authorization.Id.Exist()
}

func (authorization Authorization) Validate() bool {
	return authorization.Exist() && authorization.ExpireAT.After(time.Now())
}

const (
	endpointName = "authorizations"
	encodeFnName = "encode"
	decodeFnName = "decode"
)

func Encode(ctx context.Context, authorization Authorization) (token Token, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		bytex.FromString(endpointName), bytex.FromString(encodeFnName),
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

func Decode(ctx context.Context, token Token) (authorization Authorization, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		bytex.FromString(endpointName), bytex.FromString(decodeFnName),
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

var ErrUnauthorized = errors.Unauthorized("unauthorized")

func Validate(ctx context.Context) (err error) {
	authorization := Load(ctx)
	if authorization.Exist() {
		if authorization.Validate() {
			return
		}
		err = ErrUnauthorized
		return
	}
	r := services.LoadRequest(ctx)
	token := r.Header().Token()
	if len(token) == 0 {
		err = ErrUnauthorized
		return
	}
	authorization, err = Decode(ctx, token)
	if err != nil {
		err = ErrUnauthorized.WithCause(err).WithMeta("token", string(token))
		return
	}
	if authorization.Validate() {
		With(ctx, authorization)
		return
	}
	err = ErrUnauthorized
	return
}

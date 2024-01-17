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
	"time"
)

var (
	validateFnName = []byte("validate")
)

func ValidateContext(ctx context.Context) (err error) {
	authorization, has, loadErr := Load(ctx)
	if loadErr != nil {
		err = ErrUnauthorized.WithCause(loadErr)
		return
	}
	if has {
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
	validated, validErr := Validate(ctx, token)
	if validErr != nil {
		err = validErr
		return
	}
	With(ctx, validated)
	return
}

func Validate(ctx context.Context, token Token) (authorization Authorization, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		endpointName, validateFnName,
		token,
	)
	if handleErr != nil {
		err = handleErr
		return
	}
	authorization, err = services.ValueOfResponse[Authorization](response)
	return
}

type validateFn struct {
	encoder       TokenEncoder
	store         TokenStore
	autoRefresh   bool
	refreshWindow time.Duration
	expireTTL     time.Duration
}

func (fn *validateFn) Name() string {
	return string(validateFnName)
}

func (fn *validateFn) Internal() bool {
	return true
}

func (fn *validateFn) Readonly() bool {
	return false
}

func (fn *validateFn) Handle(r services.Request) (v any, err error) {
	param, paramErr := services.ValueOfParam[Token](r.Param())
	if paramErr != nil {
		err = errors.Warning("authorizations: invalid param")
		return
	}
	authorization, decodeErr := fn.encoder.Decode(r, param)
	if decodeErr != nil {
		err = ErrUnauthorized.WithCause(decodeErr)
		return
	}
	stored, has, getErr := fn.store.Get(r, authorization.Account, authorization.Id)
	if getErr != nil {
		err = errors.Warning("authorizations: validate token failed").WithCause(getErr)
		return
	}
	if !has {
		err = errors.Warning("authorizations: validate token failed").WithCause(fmt.Errorf("not exist"))
		return
	}
	if !stored.Validate() {
		err = ErrUnauthorized
		return
	}
	if fn.autoRefresh {
		if authorization.ExpireAT.Sub(time.Now()) < fn.refreshWindow {
			stored.ExpireAT = time.Now().Add(fn.expireTTL)
			saveErr := fn.store.Save(r, stored)
			if saveErr != nil {
				err = errors.Warning("authorizations: validate token failed at refresh").WithCause(saveErr)
				return
			}
		}
	}
	v = stored
	return
}

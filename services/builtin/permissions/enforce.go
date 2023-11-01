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

package permissions

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/authorizations"
)

var (
	ErrForbidden = errors.Forbidden("permissions: forbidden")
)

func EnforceContext(ctx context.Context, endpoint []byte, fn []byte) (err error) {
	authorization := authorizations.Load(ctx)
	if !authorization.Validate() {
		err = ErrForbidden
		return
	}
	ok, enforceErr := Enforce(ctx, EnforceParam{
		Account:  authorization.Account,
		Endpoint: endpoint,
		Fn:       fn,
	})
	if enforceErr != nil {
		err = errors.Warning("permissions: enforce failed").WithCause(enforceErr)
		return
	}
	if !ok {
		err = errors.Forbidden("permissions: forbidden")
		return
	}
	return
}

func Enforce(ctx context.Context, param EnforceParam) (ok bool, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		bytex.FromString(name), bytex.FromString(enforceFn),
		services.NewArgument(param),
	)
	if handleErr != nil {
		err = handleErr
		return
	}
	scanErr := response.Scan(&ok)
	if scanErr != nil {
		err = errors.Warning("permissions: enforce failed").WithCause(scanErr)
		return
	}
	return
}

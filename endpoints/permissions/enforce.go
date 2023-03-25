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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
)

const (
	name      = "permissions"
	enforceFn = "enforce"
)

func EnforceContext(ctx context.Context, serviceName string, fn string) (ok bool, err errors.CodeError) {
	request, hasRequest := service.GetRequest(ctx)
	if !hasRequest {
		err = errors.Warning("permissions: enforce failed").WithCause(fmt.Errorf("there is no request in context"))
		return
	}
	userId := request.User().Id()
	if !userId.Exist() {
		err = errors.Warning("permissions: enforce failed").WithCause(fmt.Errorf("there is no user id in request"))
		return
	}
	ok, err = Enforce(ctx, EnforceParam{
		UserId:  userId,
		Service: serviceName,
		Fn:      fn,
	})
	return
}

type EnforceParam struct {
	UserId  service.RequestUserId `json:"userId"`
	Service string                `json:"service"`
	Fn      string                `json:"fn"`
}

func Enforce(ctx context.Context, param EnforceParam) (ok bool, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("permissions: enforce failed").WithCause(errors.Warning("permissions: service was not deployed"))
		return
	}
	future, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, enforceFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	scanErr := future.Scan(&ok)
	if scanErr != nil {
		err = errors.Warning("permissions: enforce failed").WithCause(scanErr)
		return
	}
	return
}

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
	"github.com/aacfactory/fns/service/builtin/permissions"
	"strings"
)

func Enforce(ctx context.Context, subject string, object string, action string) (ok bool, err errors.CodeError) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		err = errors.ServiceError("permissions enforce failed").WithCause(fmt.Errorf("subject is nil"))
		return
	}
	object = strings.TrimSpace(object)
	if object == "" {
		err = errors.ServiceError("permissions enforce failed").WithCause(fmt.Errorf("object is nil"))
		return
	}
	action = strings.TrimSpace(action)
	if action == "" {
		err = errors.ServiceError("permissions enforce failed").WithCause(fmt.Errorf("action is nil"))
		return
	}
	endpoint, hasEndpoint := service.GetEndpoint(ctx, permissions.Name)
	if !hasEndpoint {
		err = errors.Warning("permissions endpoint was not found, please deploy permissions service")
		return
	}
	fr := endpoint.Request(ctx, permissions.EnforceFn, service.NewArgument(permissions.EnforceArgument{
		Subject: subject,
		Object:  object,
		Action:  action,
	}))

	result := &permissions.EnforceResult{}
	has, getResultErr := fr.Get(ctx, &result)
	if getResultErr != nil {
		err = getResultErr
		return
	}
	if !has {
		return
	}
	ok = result.Pass
	return
}

func EnforceRequest(ctx context.Context, object string, action string) (ok bool, err errors.CodeError) {
	request, hasRequest := service.GetRequest(ctx)
	if !hasRequest {
		err = errors.ServiceError("permissions enforce request failed").WithCause(fmt.Errorf("there is no request in context"))
		return
	}

	userId := request.User().Id()
	if userId == "" {
		return
	}
	ok, err = Enforce(ctx, userId, object, action)
	return
}

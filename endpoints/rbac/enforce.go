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

package rbac

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/fns/service/builtin/rbac"
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
		err = errors.ServiceError("rbac endpoint enforce failed").WithCause(fmt.Errorf("action is nil"))
		return
	}
	endpoint, hasEndpoint := service.GetEndpoint(ctx, rbac.Name)
	if !hasEndpoint {
		err = errors.Warning("rbac endpoint endpoint was not found, please deploy rbac service")
		return
	}
	fr := endpoint.Request(ctx, service.NewRequest(ctx, rbac.Name, rbac.EnforceFn, service.NewArgument(rbac.EnforceArgument{
		Subject: subject,
		Object:  object,
		Action:  action,
	})))

	result := &rbac.EnforceResult{}
	has, getResultErr := fr.Get(ctx, result)
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
		err = errors.ServiceError("rbac endpoint enforce request failed").WithCause(fmt.Errorf("there is no request in context"))
		return
	}

	userId := request.User().Id()
	if userId == "" {
		return
	}
	ok, err = Enforce(ctx, userId.String(), object, action)
	return
}

func BatchEnforceRequest(ctx context.Context, objectAndActions ...string) (ok bool, err errors.CodeError) {
	objectAndActionsLen := len(objectAndActions)
	if objectAndActionsLen == 0 || objectAndActionsLen%2 != 0 {
		err = errors.ServiceError("rbac endpoint enforce request failed").WithCause(fmt.Errorf("objects and actions are invalid"))
		return
	}
	request, hasRequest := service.GetRequest(ctx)
	if !hasRequest {
		err = errors.ServiceError("rbac endpoint enforce request failed").WithCause(fmt.Errorf("there is no request in context"))
		return
	}

	userId := request.User().Id()
	if userId == "" {
		return
	}

	for i := 0; i < objectAndActionsLen; i = i + 1 {
		ok, err = Enforce(ctx, userId.String(), objectAndActions[i], objectAndActions[i+1])
		if err != nil {
			return
		}
		if !ok {
			return
		}
	}
	return
}

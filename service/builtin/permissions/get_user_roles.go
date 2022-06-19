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
	"github.com/aacfactory/fns/service"
	"strings"
)

type GetUserRolesArgument struct {
	UserId string `json:"userId"`
}

type GetUserRolesResult struct {
	Roles []*Role `json:"roles"`
}

func userRoles(ctx context.Context, argument GetUserRolesArgument) (result *GetUserRolesResult, err errors.CodeError) {
	userId := strings.TrimSpace(argument.UserId)
	if userId == "" {
		err = errors.BadRequest("permissions: user id is empty")
		return
	}
	ps := getStore(ctx)
	roles, getErr := ps.UserRoles(ctx, userId)
	if getErr != nil {
		err = errors.ServiceError("permissions: get roles of user from store failed").WithCause(getErr)
		return
	}
	result = &GetUserRolesResult{
		Roles: roles,
	}
	return
}

func requestUserRoles(ctx context.Context) (roles []*Role, err errors.CodeError) {
	request, hasRequest := service.GetRequest(ctx)
	if !hasRequest {
		err = errors.ServiceError("permissions: there is no request in context")
		return
	}
	user := request.User()
	if !user.Authenticated() {
		err = errors.ServiceError("permissions: there is no authenticated user in context")
		return
	}
	const (
		rolesKey = "_roles_"
	)
	if user.Attributes().Contains(rolesKey) {
		roles = make([]*Role, 0, 1)
		getCachedErr := user.Attributes().Get(rolesKey, &roles)
		if getCachedErr != nil {
			err = errors.ServiceError("permissions: get roles of user attributes in context failed").WithCause(getCachedErr)
			return
		}
	} else {
		ps := getStore(ctx)
		var getErr error
		roles, getErr = ps.UserRoles(ctx, user.Id())
		if getErr != nil {
			err = errors.ServiceError("permissions: get roles of user from store failed").WithCause(getErr)
			return
		}
	}
	return
}
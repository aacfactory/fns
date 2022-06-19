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
	"strings"
)

type UserBindRolesArgument struct {
	UserId string   `json:"userId"`
	Roles  []string `json:"roles"`
}

func userBindRoles(ctx context.Context, argument UserBindRolesArgument) (err errors.CodeError) {
	userId := strings.TrimSpace(argument.UserId)
	if userId == "" {
		err = errors.BadRequest("permissions: user id is empty")
		return
	}
	roles := argument.Roles
	if roles == nil || len(roles) == 0 {
		err = errors.BadRequest("permissions: roles is empty")
		return
	}
	ps := getStore(ctx)
	bindErr := ps.UserBindRoles(ctx, userId, roles...)
	if bindErr != nil {
		err = errors.ServiceError("permissions: bind user roles failed").WithCause(bindErr)
		return
	}
	return
}

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
)

func UserUnbindRoles(ctx context.Context, userId string, roles ...string) (err errors.CodeError) {
	request, hasRequest := service.GetRequest(ctx)
	if !hasRequest {
		err = errors.Warning("permissions: user unbind roles failed").WithCause(fmt.Errorf("there is no request in context"))
		return
	}
	if !request.User().Authenticated() {
		err = errors.ServiceError("permissions: there is no authenticated user in context")
		return
	}
	endpoint, hasEndpoint := service.GetEndpoint(ctx, "permissions")
	if !hasEndpoint {
		err = errors.Warning("permissions: there is no permissions in context, please deploy permissions service")
		return
	}
	fr := endpoint.Request(ctx, "user_unbind_roles", service.NewArgument(&permissions.UserUnbindRolesArgument{
		UserId: userId,
		Roles:  roles,
	}))
	result := &service.Empty{}
	_, getResultErr := fr.Get(ctx, result)
	if getResultErr != nil {
		err = getResultErr
		return
	}
	return
}

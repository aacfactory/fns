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

func CanWriteResource(ctx context.Context, resource string) (ok bool, err errors.CodeError) {
	request, hasRequest := service.GetRequest(ctx)
	if !hasRequest {
		err = errors.Warning("permissions: verify user permissions failed").WithCause(fmt.Errorf("there is no request in context"))
		return
	}
	if !request.User().Authenticated() {
		err = errors.ServiceError("permissions: there is no authenticated user in context")
		return
	}
	userRoles, getUserRolesErr := getCurrentUserRoles(ctx)
	if getUserRolesErr != nil {
		err = errors.Warning("permissions: verify user permissions failed").WithCause(getUserRolesErr)
		return
	}
	if userRoles == nil || len(userRoles) == 0 {
		err = errors.Forbidden("permissions: forbidden").WithCause(fmt.Errorf("permissions: user has no roles"))
		return
	}
	for _, userRole := range userRoles {
		if userRole.CanWriteResource(resource) {
			return
		}
	}
	err = errors.Forbidden("permissions: forbidden").WithCause(fmt.Errorf("permissions: user can not write %s", resource))
	return
}

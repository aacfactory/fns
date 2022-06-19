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

type CheckResourcePermissionArgument struct {
	Resource string `json:"resource"`
}

type CheckResourcePermissionResult struct {
	Ok bool `json:"ok"`
}

func canReadResource(ctx context.Context, argument CheckResourcePermissionArgument) (result *CheckResourcePermissionResult, err errors.CodeError) {
	resource := strings.TrimSpace(argument.Resource)
	if resource == "" {
		err = errors.BadRequest("permissions: resource is empty")
		return
	}
	roles, getErr := requestUserRoles(ctx)
	if getErr != nil {
		err = errors.ServiceError("permissions: check user can read resource failed").WithCause(getErr)
		return
	}
	if roles == nil || len(roles) == 0 {
		result = &CheckResourcePermissionResult{
			Ok: false,
		}
		return
	}
	for _, role := range roles {
		if role.CanReadResource(resource) {
			result = &CheckResourcePermissionResult{
				Ok: true,
			}
			return
		}
	}
	result = &CheckResourcePermissionResult{
		Ok: false,
	}
	return
}

func canWriteResource(ctx context.Context, argument CheckResourcePermissionArgument) (result *CheckResourcePermissionResult, err errors.CodeError) {
	resource := strings.TrimSpace(argument.Resource)
	if resource == "" {
		err = errors.BadRequest("permissions: resource is empty")
		return
	}
	roles, getErr := requestUserRoles(ctx)
	if getErr != nil {
		err = errors.ServiceError("permissions: check user can write resource failed").WithCause(getErr)
		return
	}
	if roles == nil || len(roles) == 0 {
		result = &CheckResourcePermissionResult{
			Ok: false,
		}
		return
	}
	for _, role := range roles {
		if role.CanWriteResource(resource) {
			result = &CheckResourcePermissionResult{
				Ok: true,
			}
			return
		}
	}
	result = &CheckResourcePermissionResult{
		Ok: false,
	}
	return
}

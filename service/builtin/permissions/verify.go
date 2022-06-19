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
)

type VerifyArgument struct {
	AllowedRoles []string `json:"allowedRoles"`
}

type VerifyResult struct {
	Ok bool `json:"ok"`
}

func verify(ctx context.Context, argument VerifyArgument) (result *VerifyResult, err errors.CodeError) {
	allows := argument.AllowedRoles
	if allows == nil || len(allows) == 0 {
		err = errors.BadRequest("permissions: allowed roles is empty")
		return
	}
	roles, getErr := requestUserRoles(ctx)
	if getErr != nil {
		err = errors.ServiceError("permissions: verify failed").WithCause(getErr)
		return
	}
	if roles == nil || len(roles) == 0 {
		result = &VerifyResult{
			Ok: false,
		}
		return
	}
	for _, role := range roles {
		if role.Contains(allows) {
			result = &VerifyResult{
				Ok: true,
			}
			return
		}
	}
	result = &VerifyResult{
		Ok: false,
	}
	return
}

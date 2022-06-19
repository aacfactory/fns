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

type GetRoleArgument struct {
	Name string `json:"name"`
}

func getRole(ctx context.Context, argument GetRoleArgument) (result *Role, err errors.CodeError) {
	name := strings.TrimSpace(argument.Name)
	if name == "" {
		err = errors.BadRequest("permissions: role name is empty")
		return
	}
	ps := getStore(ctx)
	var getErr error
	result, getErr = ps.Role(ctx, name)
	if getErr != nil {
		err = errors.ServiceError("permissions: get role from store failed").WithCause(getErr)
		return
	}
	return
}

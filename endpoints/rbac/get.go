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

func GetRole(ctx context.Context, code string, withChildren bool) (v *Role, err errors.CodeError) {
	code = strings.TrimSpace(code)
	if code == "" {
		err = errors.Warning("rbac: endpoint get role failed").WithCause(fmt.Errorf("code is nil"))
		return
	}
	endpoint, hasEndpoint := service.GetEndpoint(ctx, rbac.Name)
	if !hasEndpoint {
		err = errors.Warning("rbac: endpoint endpoint was not found, please deploy rbac service")
		return
	}
	result, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, rbac.Name, rbac.RoleFn, service.NewArgument(rbac.RoleArgument{
		Code:         code,
		LoadChildren: withChildren,
	})))
	if requestErr != nil {
		err = requestErr
		return
	}
	if !result.Exist() {
		return
	}

	role := rbac.Role{}
	scanErr := result.Scan(&role)
	if scanErr != nil {
		err = errors.Warning("rbac: scan future result failed").
			WithMeta("service", rbac.Name).WithMeta("fn", rbac.RoleFn).
			WithCause(scanErr)
		return
	}
	v = newRole(&role)
	return
}

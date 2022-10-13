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

func GetRole(ctx context.Context, name string, withChildren bool) (v *Role, err errors.CodeError) {
	name = strings.TrimSpace(name)
	if name == "" {
		err = errors.ServiceError("permissions get role failed").WithCause(fmt.Errorf("name is nil"))
		return
	}
	endpoint, hasEndpoint := service.GetEndpoint(ctx, rbac.Name)
	if !hasEndpoint {
		err = errors.Warning("permissions endpoint was not found, please deploy permissions service")
		return
	}
	fr := endpoint.Request(ctx, rbac.RoleFn, service.NewArgument(rbac.RoleArgument{
		Name:         name,
		LoadChildren: withChildren,
	}))

	result := &rbac.Role{}
	has, getResultErr := fr.Get(ctx, &result)
	if getResultErr != nil {
		err = getResultErr
		return
	}
	if !has {
		return
	}
	v = newRole(result)
	return
}

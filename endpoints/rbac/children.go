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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/fns/service/builtin/rbac"
	"strings"
)

func Children(ctx context.Context, parent string, withChildren bool) (v []*Role, err errors.CodeError) {
	parent = strings.TrimSpace(parent)
	endpoint, hasEndpoint := service.GetEndpoint(ctx, rbac.Name)
	if !hasEndpoint {
		err = errors.Warning("rbac endpoint was not found, please deploy rbac service")
		return
	}
	fr := endpoint.Request(ctx, service.NewRequest(ctx, rbac.Name, rbac.ChildrenFn, service.NewArgument(rbac.ChildrenArgument{
		Parent:       parent,
		LoadChildren: withChildren,
	})))

	result := make([]*rbac.Role, 0, 1)
	has, getResultErr := fr.Get(ctx, &result)
	if getResultErr != nil {
		err = getResultErr
		return
	}
	if !has {
		return
	}
	v = make([]*Role, 0, 1)
	for _, role := range result {
		v = append(v, newRole(role))
	}
	return
}

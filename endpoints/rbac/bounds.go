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

func Bounds(ctx context.Context, subject string, flat bool) (v []*Role, err errors.CodeError) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		err = errors.Warning("rbac: endpoint list bounds role failed").WithCause(fmt.Errorf("subject is nil"))
		return
	}
	endpoint, hasEndpoint := service.GetEndpoint(ctx, rbac.Name)
	if !hasEndpoint {
		err = errors.Warning("rbac: endpoint endpoint was not found, please deploy rbac service")
		return
	}
	result, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, rbac.Name, rbac.BoundsFn, service.NewArgument(rbac.BoundsArgument{
		Subject: subject,
		Flat:    flat,
	})))
	if requestErr != nil {
		err = requestErr
		return
	}
	v = make([]*Role, 0, 1)
	if result.Exist() {
		roles := make([]*rbac.Role, 0, 1)
		scanErr := result.Scan(&roles)
		if scanErr != nil {
			err = errors.Warning("rbac: scan future result failed").
				WithMeta("service", rbac.Name).WithMeta("fn", rbac.BoundsFn).
				WithCause(scanErr)
			return
		}
		for _, role := range roles {
			v = append(v, newRole(role))
		}
	}
	return
}

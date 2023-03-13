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

func Bind(ctx context.Context, subject string, roles ...string) (err errors.CodeError) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		err = errors.Warning("rbac: endpoint bind role failed").WithCause(fmt.Errorf("subject is nil"))
		return
	}
	if roles == nil || len(roles) == 0 {
		err = errors.Warning("rbac: endpoint bind role failed").WithCause(fmt.Errorf("roles is nil"))
		return
	}
	endpoint, hasEndpoint := service.GetEndpoint(ctx, rbac.Name)
	if !hasEndpoint {
		err = errors.Warning("rbac: endpoint endpoint was not found, please deploy rbac service")
		return
	}
	fr := endpoint.Request(ctx, service.NewRequest(ctx, rbac.Name, rbac.BindFn, service.NewArgument(rbac.BindArgument{
		Subject: subject,
		Roles:   roles,
	})))

	result := &service.Empty{}
	_, getResultErr := fr.Get(ctx, result)
	if getResultErr != nil {
		err = getResultErr
		return
	}
	return
}

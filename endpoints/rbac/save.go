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

func Save(ctx context.Context, code string, name string, description string, parent string, policies []*Policy) (err errors.CodeError) {
	code = strings.TrimSpace(code)
	if code == "" {
		err = errors.Warning("rbac: endpoint save role failed").WithCause(fmt.Errorf("code is nil"))
		return
	}
	parent = strings.TrimSpace(parent)
	rolePolicies := make([]*rbac.Policy, 0, 1)
	if policies != nil {
		for _, policy := range policies {
			rolePolicies = append(rolePolicies, &rbac.Policy{
				Object: strings.TrimSpace(policy.Object),
				Action: strings.TrimSpace(policy.Action),
			})
		}
	}

	endpoint, hasEndpoint := service.GetEndpoint(ctx, rbac.Name)
	if !hasEndpoint {
		err = errors.Warning("rbac: endpoint endpoint was not found, please deploy rbac service")
		return
	}
	fr := endpoint.Request(ctx, service.NewRequest(ctx, rbac.Name, rbac.SaveFn, service.NewArgument(rbac.SaveArgument{
		Code:        code,
		Name:        name,
		Description: description,
		Parent:      parent,
		Policies:    rolePolicies,
	})))

	result := &service.Empty{}
	_, getResultErr := fr.Get(ctx, result)
	if getResultErr != nil {
		err = getResultErr
		return
	}
	return
}

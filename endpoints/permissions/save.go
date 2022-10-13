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
	"github.com/aacfactory/fns/service/builtin/permissions"
	"strings"
)

func Save(ctx context.Context, name string, parent string, policies []*Policy) (err errors.CodeError) {
	name = strings.TrimSpace(name)
	if name == "" {
		err = errors.ServiceError("permissions save role failed").WithCause(fmt.Errorf("name is nil"))
		return
	}
	parent = strings.TrimSpace(parent)
	rolePolicies := make([]*permissions.Policy, 0, 1)
	if policies != nil {
		for _, policy := range policies {
			rolePolicies = append(rolePolicies, &permissions.Policy{
				Object: strings.TrimSpace(policy.Object),
				Action: strings.TrimSpace(policy.Action),
			})
		}
	}

	endpoint, hasEndpoint := service.GetEndpoint(ctx, permissions.Name)
	if !hasEndpoint {
		err = errors.Warning("permissions endpoint was not found, please deploy permissions service")
		return
	}
	fr := endpoint.Request(ctx, permissions.SaveFn, service.NewArgument(permissions.SaveArgument{
		Name:     name,
		Parent:   parent,
		Policies: rolePolicies,
	}))

	result := &service.Empty{}
	_, getResultErr := fr.Get(ctx, &result)
	if getResultErr != nil {
		err = getResultErr
		return
	}
	return
}

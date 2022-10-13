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

import "github.com/aacfactory/fns/service/builtin/rbac"

type Policy struct {
	Object string `json:"object"`
	Action string `json:"action"`
}

func newRole(r *rbac.Role) (v *Role) {
	v = &Role{
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		Parent:      r.Parent,
		Children:    nil,
		Policies:    nil,
	}
	if r.Policies != nil && len(r.Policies) > 0 {
		v.Policies = make([]*Policy, 0, 1)
		for _, policy := range r.Policies {
			v.Policies = append(v.Policies, &Policy{
				Object: policy.Object,
				Action: policy.Action,
			})
		}
	}
	if r.Children != nil && len(r.Children) > 0 {
		v.Children = make([]*Role, 0, 1)
		for _, child := range r.Children {
			v.Children = append(v.Children, newRole(child))
		}
	}
	return
}

type Role struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Parent      string    `json:"parent"`
	Children    []*Role   `json:"children"`
	Policies    []*Policy `json:"policies"`
}

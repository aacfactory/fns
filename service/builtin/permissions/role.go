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

type Role struct {
	Name      string         `json:"name"`
	Parent    string         `json:"parent"`
	Children  []*Role        `json:"children"`
	Resources map[string]int `json:"resources"`
}

func (r *Role) AddChild(child *Role) {
	if child == nil {
		return
	}
	if child.Name == "" {
		return
	}
	if r.Children == nil {
		r.Children = make([]*Role, 0, 1)
	}
	for i, role := range r.Children {
		if role.Name == child.Name {
			r.Children[i] = child
			return
		}
	}
	r.Children = append(r.Children, child)
	return
}

func FindRole(roles []*Role, name string) (role *Role, has bool) {
	children := make([]*Role, 0, 1)
	for _, r := range roles {
		if r.Name == name {
			role = r
			has = true
			return
		}
		if r.Children != nil && len(r.Children) > 0 {
			children = append(children, r.Children...)
		}
	}
	if len(children) > 0 {
		role, has = FindRole(children, name)
	}
	return
}

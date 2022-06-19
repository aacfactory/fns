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

import "github.com/aacfactory/fns/service/builtin/permissions"

type Role interface {
	Name() string
	Children() []Role
	AddChild(child Role)
	AddReadableResource(resource string)
	AddWriteableResource(resource string)
	AddReadableAndWriteableResource(resource string)
}

func mapToPermissionRole(r Role) (v *permissions.Role) {
	v = &permissions.Role{
		Name:      r.Name(),
		Children:  make([]*permissions.Role, 0, 1),
		Resources: make(map[string]permissions.AccessKind),
	}
	if r.Children() != nil {
		for _, child := range r.Children() {
			v.AddChild(mapToPermissionRole(child))
		}
	}
	rr := r.(*role)
	if rr.Resources_ != nil {
		for resource, kind := range rr.Resources_ {
			v.Resources[resource] = permissions.AccessKind(kind)
		}
	}
	return
}

func NewRole(name string) (v Role) {
	v = &role{
		Name_:      name,
		Children_:  make([]*role, 0, 1),
		Resources_: make(map[string]int),
	}
	return
}

func newRole(r *permissions.Role) (v Role) {
	v0 := &role{
		Name_:      r.Name,
		Children_:  make([]*role, 0, 1),
		Resources_: make(map[string]int),
	}
	if r.Children != nil {
		for _, child := range r.Children {
			v0.Children_ = append(v0.Children_, newRole(child).(*role))
		}
	}
	if r.Resources != nil {
		for s, kind := range r.Resources {
			v0.Resources_[s] = int(kind)
		}
	}
	return
}

type role struct {
	Name_      string         `json:"name"`
	Children_  []*role        `json:"children"`
	Resources_ map[string]int `json:"resources"`
}

func (r *role) Name() string {
	return r.Name_
}

func (r *role) Children() []Role {
	children := make([]Role, 0, 1)
	for _, child := range r.Children_ {
		children = append(children, child)
	}
	return children
}

func (r *role) AddChild(child Role) {
	if child == nil {
		return
	}
	if child.Name() == "" {
		return
	}
	if r.Children_ == nil {
		r.Children_ = make([]*role, 0, 1)
	}
	for i, c := range r.Children_ {
		if c.Name_ == child.Name() {
			r.Children_[i] = child.(*role)
			return
		}
	}
	r.Children_ = append(r.Children_, child.(*role))
}

func (r *role) AddReadableResource(resource string) {
	if r.Resources_ == nil {
		r.Resources_ = make(map[string]int)
	}
	r.Resources_[resource] = 1
}

func (r *role) AddWriteableResource(resource string) {
	if r.Resources_ == nil {
		r.Resources_ = make(map[string]int)
	}
	r.Resources_[resource] = 2
}

func (r *role) AddReadableAndWriteableResource(resource string) {
	if r.Resources_ == nil {
		r.Resources_ = make(map[string]int)
	}
	r.Resources_[resource] = 1 | 2
}

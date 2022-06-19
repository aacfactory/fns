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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service/builtin/permissions"
)

type Role interface {
	Name() string
	Parent(ctx context.Context) (parent Role, err errors.CodeError)
	SetParent(parent Role)
	RemoveParent()
	Children() []Role
	AddChild(child Role)
	RemoveChild(child Role)
	AddReadableResource(resource string)
	AddWriteableResource(resource string)
	AddReadableAndWriteableResource(resource string)
}

func mapToPermissionRole(r Role) (v *permissions.Role) {
	rr := r.(*role)
	v = &permissions.Role{
		Name:      r.Name(),
		Parent:    rr.ParentName,
		Children:  make([]*permissions.Role, 0, 1),
		Resources: make(map[string]permissions.AccessKind),
	}
	if r.Children() != nil {
		for _, child := range r.Children() {
			v.AddChild(mapToPermissionRole(child))
		}
	}
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
		Children_:  make([]Role, 0, 1),
		Resources_: make(map[string]int),
	}
	return
}

func newRole(r *permissions.Role) (v Role) {
	v0 := &role{
		Name_:      r.Name,
		Children_:  make([]Role, 0, 1),
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
	parent     Role
	ParentName string         `json:"parent"`
	Name_      string         `json:"name"`
	Children_  []Role         `json:"children"`
	Resources_ map[string]int `json:"resources"`
}

func (r *role) Name() string {
	return r.Name_
}

func (r *role) Parent(ctx context.Context) (v Role, err errors.CodeError) {
	if r.parent != nil {
		v = r.parent
		return
	}
	if r.ParentName != "" {
		parent, getParentErr := GetRole(ctx, r.ParentName)
		if getParentErr != nil {
			err = errors.ServiceError("permissions: get parent role failed").WithMeta("role", r.Name_).WithMeta("parent", r.ParentName)
			return
		}
		r.SetParent(parent)
		v = parent
		return
	}
	return
}

func (r *role) SetParent(parent Role) {
	r.parent = parent
	r.ParentName = parent.Name()
}

func (r *role) RemoveParent() {
	r.ParentName = ""
	r.parent = nil
}

func (r *role) Children() []Role {
	return r.Children_
}

func (r *role) AddChild(child Role) {
	if child == nil {
		return
	}
	if child.Name() == "" {
		return
	}
	if r.Children_ == nil {
		r.Children_ = make([]Role, 0, 1)
	}
	for i, c := range r.Children_ {
		if c.Name() == child.Name() {
			r.Children_[i] = child.(*role)
			return
		}
	}
	r.Children_ = append(r.Children_, child.(*role))
}

func (r *role) RemoveChild(child Role) {
	if child == nil {
		return
	}
	if r.Children_ == nil || len(r.Children_) == 0 {
		return
	}
	if len(r.Children_) == 0 {
		r.Children_ = nil
		return
	}
	children := make([]Role, 0, 1)
	for _, c := range r.Children_ {
		if c.Name() != child.Name() {
			children = append(children, c)
		}
	}
	r.Children_ = children
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

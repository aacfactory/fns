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

const (
	R  = AccessKind(1)
	W  = AccessKind(2)
	RW = R | W
)

type AccessKind int

type Role struct {
	Name      string                `json:"name"`
	Parent    string                `json:"parent"`
	Children  []*Role               `json:"children"`
	Resources map[string]AccessKind `json:"resources"`
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

func (r *Role) AddResource(resource string, kind AccessKind) {
	if resource == "" {
		return
	}
	if kind&RW == 0 {
		return
	}
	if r.Resources == nil {
		r.Resources = make(map[string]AccessKind)
	}
	r.Resources[resource] = kind
	return
}

func (r *Role) CanReadResource(resource string) (ok bool) {
	if r.Resources == nil {
		return
	}
	kind, has := r.Resources[resource]
	if !has {
		return
	}
	ok = kind&R != 0
	return
}

func (r *Role) CanWriteResource(resource string) (ok bool) {
	if r.Resources == nil {
		return
	}
	kind, has := r.Resources[resource]
	if !has {
		return
	}
	ok = kind&W != 0
	return
}

func (r *Role) CanReadAndResourceResource(resource string) (ok bool) {
	if r.Resources == nil {
		return
	}
	kind, has := r.Resources[resource]
	if !has {
		return
	}
	ok = kind&RW != 0
	return
}

func (r *Role) Contains(roles []string) (ok bool) {
	for _, role := range roles {
		if r.Name == role {
			ok = true
			return
		}
		if r.Children != nil && len(r.Children) > 0 {
			for _, child := range r.Children {
				ok = child.Contains(roles)
				if ok {
					return
				}
			}
		}
	}
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

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

package fns

import (
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
)

type fakePermissions struct{}

func (f *fakePermissions) Validate(ctx Context, namespace string, fn string) (err errors.CodeError) {
	err = errors.Warning("fns Permissions: permissions was not enabled, please use fns.RegisterPermissionsDefinitionsLoaderRetriever() to setup and enable in services config")
	return
}

func (f *fakePermissions) SaveUserRoles(ctx Context, roles ...string) (err errors.CodeError) {
	err = errors.Warning("fns Permissions: permissions was not enabled, please use fns.RegisterPermissionsDefinitionsLoaderRetriever() to setup and enable in services config")
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

var permissionsDefinitionsLoaderRetrieverMap = make(map[string]PermissionsDefinitionsLoaderRetriever)

type PermissionsDefinitionsLoaderRetriever func(config configuares.Raw) (loader PermissionsDefinitionsLoader, err error)

func RegisterPermissionsDefinitionsLoaderRetriever(kind string, retriever PermissionsDefinitionsLoaderRetriever) {
	permissionsDefinitionsLoaderRetrieverMap[kind] = retriever
}

// Permissions
// 基于RBAC的权限控制器
// 角色：角色树，控制器不存储用户的角色。
// 资源：fn
// 控制：是否可以使用（不可以使用优先于可以使用）
type Permissions interface {
	// Validate 验证当前 context 中 user 对 fn 的权限
	Validate(ctx Context, namespace string, fn string) (err errors.CodeError)
	// SaveUserRoles 将角色保存到 当前 context 的 user attributes 中
	SaveUserRoles(ctx Context, roles ...string) (err errors.CodeError)
}

type PermissionsDefinitions struct {
	data map[string]map[string]bool
}

func (d *PermissionsDefinitions) Add(namespace string, fn string, role string, accessible bool) {
	if namespace == "" || fn == "" || role == "" {
		return
	}
	if d.data == nil {
		d.data = make(map[string]map[string]bool)
	}
	key := fmt.Sprintf("%s:%s", namespace, fn)
	g, has := d.data[key]
	if !has {
		g = make(map[string]bool)
	}
	g[role] = accessible
	d.data[key] = g
}
func (d *PermissionsDefinitions) Accessible(namespace string, fn string, roles []string) (accessible bool) {
	if namespace == "" || fn == "" || d.data == nil || len(d.data) == 0 {
		return
	}
	key := fmt.Sprintf("%s:%s", namespace, fn)
	g, has := d.data[key]
	if !has {
		accessible = false
		return
	}
	_, all := g["*"]
	if all {
		accessible = true
		return
	}
	not := false
	n := 0
	for _, role := range roles {
		x, hasRole := g[role]
		if !hasRole {
			continue
		}
		if !x {
			not = true
			break
		}
		n++
	}
	if not {
		accessible = false
		return
	}
	accessible = n > 0
	return
}

// PermissionsDefinitionsLoader
// 存储权限设定的加载器
type PermissionsDefinitionsLoader interface {
	Load() (definitions *PermissionsDefinitions, err errors.CodeError)
}

// +-------------------------------------------------------------------------------------------------------------------+

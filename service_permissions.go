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

const (
	userAttributeRoleKey = "roles"
)

// +-------------------------------------------------------------------------------------------------------------------+

func newRbacPermissions(loader PermissionsDefinitionsLoader) (p *rbacPermissions, err error) {

	definitions, loadErr := loader.Load()
	if loadErr != nil {
		err = fmt.Errorf("fns Permissions: create failed for load definitions, %v", loadErr)
		return
	}
	if definitions == nil {
		definitions = &PermissionsDefinitions{
			data: make(map[string]map[string]bool),
		}
	}

	p = &rbacPermissions{
		definitions: definitions,
	}

	return
}

type rbacPermissions struct {
	definitions *PermissionsDefinitions
}

func (p *rbacPermissions) Validate(ctx Context, namespace string, fn string) (err errors.CodeError) {
	if !ctx.User().Exists() {
		err = errors.Forbidden(fmt.Sprintf("fns Permissions: %s/%s is not accessible", namespace, fn))
		return
	}
	if !ctx.User().Attributes().Contains(userAttributeRoleKey) {
		err = errors.Forbidden(fmt.Sprintf("fns Permissions: %s/%s is not accessible", namespace, fn))
		return
	}
	roles := make([]string, 0, 1)
	getErr := ctx.User().Attributes().Get(userAttributeRoleKey, &roles)
	if getErr != nil {
		err = errors.Forbidden(fmt.Sprintf("fns Permissions: %s/%s is not accessible", namespace, fn)).WithCause(getErr)
		return
	}

	if !p.definitions.Accessible(namespace, fn, roles) {
		err = errors.Forbidden(fmt.Sprintf("fns Permissions: %s/%s is not accessible", namespace, fn))
		return
	}

	return
}

func (p *rbacPermissions) SaveUserRoles(ctx Context, roles ...string) (err errors.CodeError) {
	if !ctx.User().Exists() {
		err = errors.Forbidden("fns Permissions: save user roles failed for no user")
		return
	}
	if roles == nil || len(roles) == 0 {
		_ = ctx.User().Attributes().Remove(userAttributeRoleKey)
		return
	}
	setErr := ctx.User().Attributes().Put(userAttributeRoleKey, &roles)
	if setErr != nil {
		err = errors.Forbidden("fns Permissions: save user roles failed for put into user attributes").WithCause(setErr)
		return
	}
	return
}

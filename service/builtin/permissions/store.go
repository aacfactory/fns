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
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/wildcard"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/logs"
)

type PolicyRecord struct {
	Object string `json:"object"`
	Action string `json:"action"`
}

type RoleRecord struct {
	Name     string          `json:"name"`
	Parent   string          `json:"parent"`
	Policies []*PolicyRecord `json:"policies"`
}

func (r *RoleRecord) mapToRole() (v *Role) {
	v = &Role{
		Name:     r.Name,
		Parent:   r.Parent,
		Children: nil,
		Policies: nil,
	}
	if r.Policies != nil && len(r.Policies) > 0 {
		policies := make([]*Policy, 0, len(r.Policies))
		for _, policy := range r.Policies {
			if policy.Action == "" {
				policy.Action = "*"
			}
			policies = append(policies, &Policy{
				Object:  policy.Object,
				Action:  policy.Action,
				matcher: wildcard.New(policy.Action),
			})
		}
		v.Policies = policies
	}
	return
}

type StoreOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Store interface {
	Build(options StoreOptions) (err error)
	Role(ctx context.Context, name string) (role *RoleRecord, err error)
	Roles(ctx context.Context) (roles []*RoleRecord, err error)
	Children(ctx context.Context, parent string) (children []*RoleRecord, err error)
	SaveRole(ctx context.Context, role *RoleRecord) (err error)
	RemoveRole(ctx context.Context, role *RoleRecord) (err error)
	Binds(ctx context.Context, subject string) (roles []*RoleRecord, err error)
	Bind(ctx context.Context, subject string, roles []*RoleRecord) (err error)
	Unbind(ctx context.Context, subject string, roles []*RoleRecord) (err error)
	Close() (err error)
}

type storeComponent struct {
	store Store
}

func (component *storeComponent) Name() (name string) {
	name = "store"
	return
}

func (component *storeComponent) Build(options service.ComponentOptions) (err error) {
	err = component.store.Build(StoreOptions{
		Log:    options.Log,
		Config: options.Config,
	})
	return
}

func (component *storeComponent) Close() {
	_ = component.store.Close()
}

func getStore(ctx context.Context) (v Store) {
	c, has := service.GetComponent(ctx, "store")
	if !has {
		panic(fmt.Sprintf("%+v", errors.Warning("permissions: there is no store in context")))
	}
	sc, ok := c.(*storeComponent)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("permissions: type of store in context is invalid")))
	}
	v = sc.store
	return
}

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
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/logs"
)

type StoreOptions struct {
	Log    logs.Logger
	Config configuares.Config
}

type Store interface {
	Build(options StoreOptions) (err error)
	GetUserRoles()
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

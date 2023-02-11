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

package cluster

import (
	"context"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/service"
)

type ManagementBuilder func(config configures.Config) (management Management, err error)

type Management interface {
	Join(ctx context.Context, endpoints service.Endpoints) (err error)
	Leave(ctx context.Context) (err error)
	Barrier() (barrier service.Barrier)
	Discovery() (discovery service.EndpointDiscovery)
	Shared() (shared service.Shared)
}

var (
	managements = make(map[string]ManagementBuilder)
)

func RegisterManagementBuilder(name string, builder ManagementBuilder) {
	managements[name] = builder
}

func GetManagementBuilder(name string) (builder ManagementBuilder, has bool) {
	builder, has = managements[name]
	return
}

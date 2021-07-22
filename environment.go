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
	"github.com/aacfactory/cluster"
)

type Environment interface {
	ClusterMode() (ok bool)
	Config() (config Config)
	Discovery() (discovery cluster.ServiceDiscovery)
}

func newFnsEnvironment(config Config, discovery cluster.ServiceDiscovery) Environment {
	return &fnsEnvironment{
		config:    config,
		discovery: discovery,
	}
}

type fnsEnvironment struct {
	config    Config
	discovery cluster.ServiceDiscovery
}

func (env *fnsEnvironment) ClusterMode() (ok bool) {
	ok = env.discovery != nil
	return
}

func (env *fnsEnvironment) Config() (config Config) {
	config = env.config
	return
}

func (env *fnsEnvironment) Discovery() (discovery cluster.ServiceDiscovery) {
	if env.discovery == nil {
		panic(fmt.Errorf("fns is not in cluster mode"))
	}
	discovery = env.discovery
	return
}

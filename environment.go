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
)

type Environment interface {
	ClusterMode() (ok bool)
	Config() (config Config)
	Discovery() (dc Discovery)
}

func newFnsEnvironment(config Config, dc Discovery) Environment {
	return &fnsEnvironment{
		clusterMode: dc != nil,
		config:      config,
		discovery:   dc,
	}
}

type fnsEnvironment struct {
	clusterMode bool
	config      Config
	discovery   Discovery
}

func (env *fnsEnvironment) ClusterMode() (ok bool) {
	ok = env.clusterMode
	return
}

func (env *fnsEnvironment) Config() (config Config) {
	config = env.config
	return
}

func (env *fnsEnvironment) Discovery() (dc Discovery) {
	if !env.ClusterMode() {
		panic(fmt.Errorf("fns is not in cluster mode"))
	}
	dc = env.discovery
	return
}

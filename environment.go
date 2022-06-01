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
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/logs"
)

type Environments interface {
	AppId() (id string)
	Version() (v string)
	Running() (ok bool)
	Config(name string) (config configuares.Config, has bool)
	Log() (log logs.Logger)
}

func newEnvironments(appId string, version string, running *commons.SafeFlag, config configuares.Config, log logs.Logger) *environments {
	return &environments{appId: appId, version: version, running: running, config: config, log: log}
}

type environments struct {
	appId   string
	version string
	running *commons.SafeFlag
	config  configuares.Config
	log     logs.Logger
}

func (env *environments) AppId() (id string) {
	id = env.appId
	return
}

func (env *environments) Version() (v string) {
	v = env.version
	return
}

func (env *environments) Running() (ok bool) {
	ok = env.running.IsOn()
	return
}

func (env *environments) Config(name string) (config configuares.Config, has bool) {
	config, has = env.config.Node(name)
	return
}

func (env *environments) Log() (log logs.Logger) {
	log = env.log
	return
}

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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/logs"
	"strings"
)

type BootstrapOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Bootstrap interface {
	Build(options BootstrapOptions) (err error)
	Id() (id string)
	Ip() (ip string)
	FindMembers(ctx context.Context) (addresses []string)
}

var (
	registeredBootstraps = map[string]Bootstrap{"members": &defaultBootstrap{}}
)

func RegisterBootstrap(kind string, bootstrap Bootstrap) (ok bool) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return
	}
	if bootstrap == nil {
		return
	}
	_, has := registeredBootstraps[kind]
	if has {
		return
	}
	registeredBootstraps[kind] = bootstrap
	ok = true
	return
}

func getRegisteredBootstrap(kind string) (bootstrap Bootstrap, has bool) {
	bootstrap, has = registeredBootstraps[kind]
	return
}

type defaultBootstrap struct {
	id      string
	ip      string
	members []string
}

func (b *defaultBootstrap) Build(options BootstrapOptions) (err error) {
	members := make([]string, 0, 1)
	getErr := options.Config.As(&members)
	if getErr != nil {
		err = errors.Warning("fns: members is undefined in cluster.options config")
		return
	}
	if len(members) == 0 {
		err = errors.Warning("fns: members is undefined in cluster.options config")
		return
	}
	b.members = members
	b.id = uid.UID()
	b.ip = commons.GetGlobalUniCastIpFromHostname()
	if b.ip == "" {
		err = errors.Warning("can not get ip from hostname, please set FNS_IP into system env")
		return
	}
	return
}

func (b *defaultBootstrap) Id() (id string) {
	id = b.id
	return
}

func (b *defaultBootstrap) Ip() (ip string) {
	ip = b.ip
	return
}

func (b *defaultBootstrap) FindMembers(_ context.Context) (addresses []string) {
	addresses = b.members
	return
}

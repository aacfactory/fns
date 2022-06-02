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
	sc "context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
	"strings"
	"time"
)

type Manager struct {
	log               logs.Logger
	bootstrap         Bootstrap
	heartbeatDuration time.Duration
	heartbeatStopCh   chan struct{}
	node              *Node
	client            Client
	resources         Resources
	registrations     *RegistrationsManager
}

type ManagerOptions struct {
	Log           logs.Logger
	Port          int
	Config        *Config
	ClientTLS     *tls.Config
	ClientBuilder ClientBuilder
}

func NewManager(options ManagerOptions) (manager *Manager, err error) {
	config := options.Config
	kind := strings.TrimSpace(config.Kind)
	if kind == "" {
		err = errors.Warning("fns: kind is undefined")
		return
	}
	if config.Options == nil || len(config.Options) == 0 {
		err = errors.Warning("fns: options is undefined")
		return
	}
	log := options.Log.With("fns", "cluster")
	client, clientErr := options.ClientBuilder(newClientOptions(log.With("cluster", "client"), options.ClientTLS, config.Client))
	if clientErr != nil {
		err = errors.Warning("fns: build client failed").WithCause(clientErr)
		return
	}
	bootstrap, hasBootstrap := getRegisteredBootstrap(kind)
	if !hasBootstrap {
		err = errors.Warning(fmt.Sprintf("fns: %s kind bootstrap is not registerd", kind))
		return
	}
	bootstrapConfig, bootstrapConfigErr := configuares.NewJsonConfig(config.Options)
	if bootstrapConfigErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: options is invalid")).WithCause(bootstrapConfigErr)
		return
	}
	bootstrapBuildErr := bootstrap.Build(BootstrapOptions{
		Config: bootstrapConfig,
		Log:    log.With("cluster", "bootstrap"),
	})
	if bootstrapBuildErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: build bootstrap failed")).WithCause(bootstrapBuildErr)
		return
	}
	id := bootstrap.Id()
	if id == "" {
		err = fmt.Errorf("fns: can not get my id from bootstrap")
		return
	}
	ip := bootstrap.Ip()
	if ip == "" {
		err = fmt.Errorf("fns: can not get my ip from bootstrap")
		return
	}
	manager = &Manager{
		log:               log,
		bootstrap:         bootstrap,
		heartbeatDuration: 0,
		heartbeatStopCh:   make(chan struct{}, 1),
		node: &Node{
			Id:               id,
			SSL:              options.ClientTLS != nil,
			Address:          fmt.Sprintf("%s:%d", ip, options.Port),
			Services:         make([]string, 0, 1),
			InternalServices: make([]string, 0, 1),
			client:           client,
		},
		client:        client,
		resources:     newResourcesManager(log, client),
		registrations: newRegistrationsManager(log),
	}
	return
}

func (manager *Manager) Join(ctx sc.Context) {
	//
}

func (manager *Manager) Leave(ctx sc.Context) {
	//
}

func (manager *Manager) Node() (node *Node) {
	node = manager.node
	return
}

func (manager *Manager) Resources() (resources Resources) {
	resources = manager.resources
	return
}

func (manager *Manager) Registrations() (registrations *RegistrationsManager) {
	registrations = manager.registrations
	return
}

func (manager *Manager) push(ctx sc.Context) {

	return
}

func (manager *Manager) heartbeat() {

	return
}

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
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"strings"
	"time"
)

type Manager struct {
	log                 logs.Logger
	bootstrap           Bootstrap
	checkHealthDuration time.Duration
	checkHealthCancel   func()
	node                *Node
	client              Client
	registrations       *RegistrationsManager
}

type ManagerOptions struct {
	Log                 logs.Logger
	Port                int
	Kind                string
	CheckHealthDuration time.Duration
	Options             json.RawMessage
	Client              Client
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
	checkHealthSec := config.CheckHealthSecond
	if checkHealthSec < 1 {
		checkHealthSec = 60
	}
	manager = &Manager{
		log:                 log,
		bootstrap:           bootstrap,
		checkHealthDuration: time.Duration(checkHealthSec) * time.Second,
		node: &Node{
			Id:               id,
			SSL:              options.ClientTLS != nil,
			Address:          fmt.Sprintf("%s:%d", ip, options.Port),
			Services:         make([]string, 0, 1),
			InternalServices: make([]string, 0, 1),
			client:           client,
		},
		client:        client,
		registrations: newRegistrationsManager(log),
	}
	return
}

func (manager *Manager) Join(ctx context.Context) {
	//

	ctx0, cancel := context.WithCancel(ctx)
	manager.checkHealthCancel = cancel
	go manager.checkHealth(ctx0)
}

func (manager *Manager) Leave(ctx context.Context) {
	manager.checkHealthCancel()
	//
}

func (manager *Manager) Node() (node *Node) {
	node = manager.node
	return
}

func (manager *Manager) Registrations() (registrations *RegistrationsManager) {
	registrations = manager.registrations
	return
}

func (manager *Manager) checkHealth(ctx context.Context) {
	for {
		stopped := false
		select {
		case <-ctx.Done():
			stopped = true
			break
		case <-time.After(manager.checkHealthDuration):
			// members

			// boostrap
			manager.bootstrap.FindMembers(ctx)
		}
		if stopped {
			break
		}
	}
	return
}

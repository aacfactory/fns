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

type ManagerOptions struct {
	Log     logs.Logger
	Port    int
	Kind    string
	Options json.RawMessage
	Client  Client
}

func NewManager(options ManagerOptions) (manager *Manager, err error) {
	kind := strings.TrimSpace(options.Kind)
	if kind == "" {
		err = errors.Warning("fns: kind is undefined")
		return
	}
	log := options.Log.With("fns", "cluster")
	bootstrap, hasBootstrap := getRegisteredBootstrap(kind)
	if !hasBootstrap {
		err = errors.Warning(fmt.Sprintf("fns: %s kind bootstrap is not registerd", kind))
		return
	}
	bootstrapConfig, bootstrapConfigErr := configuares.NewJsonConfig(options.Options)
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
		log:       log.With("cluster", "manager"),
		bootstrap: bootstrap,
		interval:  60 * time.Second,
		node: &Node{
			Id:               id,
			Address:          fmt.Sprintf("%s:%d", ip, options.Port),
			Services:         make([]string, 0, 1),
			InternalServices: make([]string, 0, 1),
			client:           options.Client,
		},
		client:        options.Client,
		registrations: newRegistrationsManager(log),
		stopCh:        make(chan struct{}, 1),
	}
	return
}

type Manager struct {
	log           logs.Logger
	bootstrap     Bootstrap
	interval      time.Duration
	node          *Node
	client        Client
	registrations *RegistrationsManager
	stopCh        chan struct{}
}

func (manager *Manager) Join() {
	go func(manager *Manager) {
		for {
			stopped := false
			select {
			case <-manager.stopCh:
				stopped = true
				break
			case <-time.After(manager.interval):
				// members

				// boostrap
				manager.bootstrap.FindMembers(context.TODO())

			}
			if stopped {
				break
			}
		}
	}(manager)
}

func (manager *Manager) Leave() {
	close(manager.stopCh)

	manager.registrations.Close()
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

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
	"github.com/aacfactory/logs"
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

func NewManager(log logs.Logger, bootstrap Bootstrap, client Client) *Manager {
	return &Manager{
		log:               log,
		bootstrap:         bootstrap,
		heartbeatDuration: 0,
		heartbeatStopCh:   nil,
		node:              nil,
		client:            client,
		resources:         nil,
		registrations:     nil,
	}
}

func (manager *Manager) Join(ctx sc.Context, ssl bool, services []string, internals []string) {
	manager.node.SSL = ssl
	manager.node.Services = services
	manager.node.InternalServices = internals
	//
}

func (manager *Manager) Leave(ctx sc.Context) {
	//
}

func (manager *Manager) Members() (nodes []*Node) {

	return
}

func (manager *Manager) Resources() (resources Resources) {

	return
}

func (manager *Manager) Registrations() (registrations *RegistrationsManager) {

	return
}

func (manager *Manager) push(ctx sc.Context) {

	return
}

func (manager *Manager) heartbeat() {

	return
}

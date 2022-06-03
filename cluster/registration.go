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
	"fmt"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"sync"
	"sync/atomic"
)

type Registration struct {
	Id               string
	Name             string
	Internal         bool
	Address          string
	SSL              bool
	client           Client
	unavailableTimes int64
}

func (r *Registration) Request(ctx sc.Context, fn string, header http.Header, body []byte) (respBody []byte, err error) {
	schema := ""
	if r.SSL {
		schema = "https"
	} else {
		schema = "http"
	}
	url := fmt.Sprintf("%s://%s/%s/%s", schema, r.Address, r.Name, fn)
	respBody, err = r.client.Do(ctx, http.MethodPost, url, header, body)
	return
}

func (r *Registration) Key() (key string) {
	key = r.Id
	return
}

func (r *Registration) AddUnavailableTimes() {
	atomic.AddInt64(&r.unavailableTimes, 1)
	return
}

func (r *Registration) Unavailable() (ok bool) {
	ok = atomic.LoadInt64(&r.unavailableTimes) > 5
	return
}

func newRegistrations(value *Registration) *Registrations {
	return &Registrations{
		r: commons.NewRing(value),
	}
}

type Registrations struct {
	r *commons.Ring
}

func (r *Registrations) Next() (v *Registration, has bool) {
	p := r.r.Next()
	if p == nil {
		return
	}
	v, has = p.(*Registration)
	return
}

func (r *Registrations) Append(v *Registration) {
	r.r.Append(v)
	return
}

func (r *Registrations) Remove(v *Registration) {
	r.r.Remove(v)
	return
}

func (r *Registrations) Size() (size int) {
	size = r.r.Size()
	return
}

func (r *Registrations) Get(id string) (v *Registration, has bool) {
	if id == "" {
		return
	}
	p := r.r.Get(id)
	if p == nil {
		return
	}
	v, has = p.(*Registration)
	return
}

func newRegistrationsManager(log logs.Logger) *RegistrationsManager {
	return &RegistrationsManager{
		log:       log.With("cluster", "registrations"),
		mutex:     sync.Mutex{},
		nodes:     sync.Map{},
		values:    sync.Map{},
		resources: sync.Map{},
	}
}

type RegistrationsManager struct {
	log       logs.Logger
	mutex     sync.Mutex
	nodes     sync.Map
	values    sync.Map
	resources sync.Map
}

func (manager *RegistrationsManager) members() (values []*Node) {
	values = make([]*Node, 0, 1)
	manager.nodes.Range(func(_, value interface{}) bool {
		node := value.(*Node)
		values = append(values, node)
		return true
	})
	return
}

func (manager *RegistrationsManager) containsMember(node *Node) (ok bool) {
	_, ok = manager.nodes.Load(node.Id)
	return
}

func (manager *RegistrationsManager) register(node *Node) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	_, hasNode := manager.nodes.Load(node.Id)
	if hasNode {
		return
	}
	registrations := node.Registrations()
	for _, registration := range registrations {
		value, has := manager.values.Load(registration.Name)
		if has {
			registered := value.(*Registrations)
			_, exist := registered.Get(registration.Id)
			if exist {
				continue
			}
			registered.Append(registration)
		} else {
			manager.values.Store(registration.Name, newRegistrations(registration))
		}
	}
	return
}

func (manager *RegistrationsManager) deregister(node *Node) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	existNode0, hasNode := manager.nodes.Load(node.Id)
	if !hasNode {
		return
	}
	existNode := existNode0.(*Node)
	registrations := existNode.Registrations()
	for _, registration := range registrations {
		value, has := manager.values.Load(registration.Name)
		if !has {
			continue
		}
		registered := value.(*Registrations)
		existed, exist := registered.Get(registration.Id)
		if !exist {
			continue
		}
		registered.Remove(existed)
		if registered.Size() == 0 {
			manager.values.Delete(registration.Name)
		}
	}
	existNode.resources.Range(func(key, _ interface{}) bool {
		manager.resources.Delete(key)
		existNode.resources.Delete(key)
		return true
	})
	return
}

func (manager *RegistrationsManager) GetRegistrations(name string) (registrations *Registrations, has bool) {
	value, exist := manager.values.Load(name)
	if !exist {
		return
	}
	registrations = value.(*Registrations)
	has = true
	return
}

func (manager *RegistrationsManager) GetRegistration(name string, registrationId string) (registration *Registration, has bool) {
	value, exist := manager.values.Load(name)
	if !exist {
		return
	}
	registered := value.(*Registrations)
	for {
		registration, has = registered.Next()
		if !has {
			break
		}
		if registration.Id == registrationId {
			break
		}
	}
	return
}

func (manager *RegistrationsManager) RemoveUnavailableRegistration(name string, registrationId string) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	value, exist := manager.values.Load(name)
	if !exist {
		return
	}
	registered := value.(*Registrations)
	registered.Remove(&Registration{Id: registrationId})
}

func (manager *RegistrationsManager) GetNodeResource(key string) (value []byte, has bool) {
	nodeId, nodeIdLoaded := manager.resources.Load(key)
	if !nodeIdLoaded {
		return
	}
	node0, nodeLoaded := manager.nodes.Load(nodeId)
	if !nodeLoaded {
		return
	}
	node := node0.(*Node)
	value0, valueLoaded := node.resources.Load(key)
	if !valueLoaded {
		manager.resources.Delete(key)
		return
	}
	value, has = value0.([]byte)
	return
}

func (manager *RegistrationsManager) getNodeResources(nodeId string) (v map[string]json.RawMessage) {
	node0, nodeLoaded := manager.nodes.Load(nodeId)
	if !nodeLoaded {
		return
	}
	node := node0.(*Node)
	node.resources.Range(func(key0, value0 interface{}) bool {
		key := key0.(string)
		value := value0.([]byte)
		v[key] = value
		return true
	})
	return
}

func (manager *RegistrationsManager) setNodeResource(nodeId string, key string, value []byte) {
	node0, nodeLoaded := manager.nodes.Load(nodeId)
	if !nodeLoaded {
		return
	}
	node := node0.(*Node)
	node.resources.Store(key, value)
	manager.resources.Store(key, node.Id)
	return
}

func (manager *RegistrationsManager) delNodeResource(key string) {
	nodeId, nodeIdLoaded := manager.resources.Load(key)
	if !nodeIdLoaded {
		return
	}
	manager.resources.Delete(key)
	node0, nodeLoaded := manager.nodes.Load(nodeId)
	if !nodeLoaded {
		return
	}
	node := node0.(*Node)
	node.resources.Delete(key)
	return
}

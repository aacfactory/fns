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
	"crypto/tls"
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/logs"
	"sync"
	"sync/atomic"
)

type Registration struct {
	Id               string
	Name             string
	Internal         bool
	Address          string
	SSL              bool
	unavailableTimes int64
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

type RegistrationsManager struct {
	log       logs.Logger
	mutex     sync.Mutex
	clientTLS *tls.Config
	nodes     sync.Map
	values    sync.Map
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

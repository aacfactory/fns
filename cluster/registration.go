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
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/logs"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Registration struct {
	Id               string
	Name             string
	Internal         bool
	Address          string
	SSL              bool
	client           Client
	unavailableTimes int64
	checkLock        sync.Mutex
	lastCheckTime    time.Time
}

func (r *Registration) Key() (key string) {
	key = r.Id
	return
}

func (r *Registration) Request(ctx context.Context, fn string, argument service.Argument) (result service.Result) {
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

func (r *Registration) checkHealth() (ok bool) {

	return
}

func (r *Registration) addUnavailableTimes() {
	atomic.AddInt64(&r.unavailableTimes, 1)
	return
}

func (r *Registration) resetUnavailableTimes() {
	atomic.StoreInt64(&r.unavailableTimes, 0)
	return
}

func (r *Registration) Unavailable() (ok bool) {
	ok = atomic.LoadInt64(&r.unavailableTimes) > 5
	if ok {
		r.checkLock.Lock()
		if time.Now().Sub(r.lastCheckTime) < 10*time.Second {
			r.checkLock.Unlock()
			return
		}
		r.lastCheckTime = time.Now()
		r.checkLock.Unlock()
		if r.checkHealth() {
			r.resetUnavailableTimes()
		}
	}
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
	return
}

func (manager *RegistrationsManager) Get(_ context.Context, name string) (endpoint service.Endpoint, has bool) {
	value, exist := manager.values.Load(name)
	if !exist {
		return
	}
	registered := value.(*Registrations)
	for i := 0; i < 5; i++ {
		registration, ok := registered.Next()
		if !ok {
			return
		}
		if registration.Unavailable() {
			continue
		}
		endpoint = registration
		has = true
		return
	}
	return
}

func (manager *RegistrationsManager) GetExact(_ context.Context, name string, id string) (endpoint service.Endpoint, has bool) {
	value, exist := manager.values.Load(name)
	if !exist {
		return
	}
	registered := value.(*Registrations)
	for {
		registration, ok := registered.Next()
		if !ok {
			return
		}
		if registration.Id == id {
			if registration.Unavailable() {
				return
			}
			endpoint = registration
			has = true
			return
		}
	}
}

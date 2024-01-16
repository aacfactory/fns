/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package clusters

import (
	"fmt"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/commons/window"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func NewEndpoint(log logs.Logger, address string, id string, version versions.Version, name string, internal bool, document documents.Endpoint, client transports.Client, signature signatures.Signature) (endpoint *Endpoint) {
	endpoint = &Endpoint{
		log: log.With("endpoint", name),
		info: services.EndpointInfo{
			Id:        id,
			Version:   version,
			Address:   address,
			Name:      name,
			Internal:  internal,
			Functions: nil,
			Document:  document,
		},
		running:   atomic.Bool{},
		functions: make(services.Fns, 0, 1),
		client:    client,
		signature: signature,
		errs:      window.NewTimes(10 * time.Second),
	}
	endpoint.running.Store(true)
	return
}

type Endpoint struct {
	log       logs.Logger
	info      services.EndpointInfo
	running   atomic.Bool
	functions services.Fns
	client    transports.Client
	signature signatures.Signature
	errs      *window.Times
}

func (endpoint *Endpoint) Running() bool {
	return endpoint.running.Load()
}

func (endpoint *Endpoint) Id() string {
	return endpoint.info.Id
}

func (endpoint *Endpoint) Address() string {
	return endpoint.info.Address
}

func (endpoint *Endpoint) Name() string {
	return endpoint.info.Name
}

func (endpoint *Endpoint) Internal() bool {
	return endpoint.info.Internal
}

func (endpoint *Endpoint) Document() documents.Endpoint {
	return endpoint.info.Document
}

func (endpoint *Endpoint) Functions() services.Fns {
	return endpoint.functions
}

func (endpoint *Endpoint) Shutdown(_ context.Context) {
	endpoint.running.Store(false)
	endpoint.client.Close()
}

func (endpoint *Endpoint) IsHealth() bool {
	return endpoint.errs.Value() < 5
}

func (endpoint *Endpoint) AddFn(name string, internal bool, readonly bool) {
	fn := &Fn{
		log:          endpoint.log.With("fn", name),
		endpointName: endpoint.info.Name,
		address:      endpoint.info.Address,
		name:         name,
		internal:     internal,
		readonly:     readonly,
		path:         bytex.FromString(fmt.Sprintf("/%s/%s", endpoint.info.Name, name)),
		signature:    endpoint.signature,
		errs:         endpoint.errs,
		health:       atomic.Bool{},
		client:       endpoint.client,
	}
	fn.health.Store(true)
	endpoint.functions = endpoint.functions.Add(fn)
	endpoint.info.Functions = append(endpoint.info.Functions, services.FnInfo{
		Name:     name,
		Readonly: readonly,
		Internal: internal,
	})
	sort.Sort(endpoint.info.Functions)
}

func (endpoint *Endpoint) Info() services.EndpointInfo {
	return endpoint.info
}

type VersionEndpoints struct {
	version versions.Version
	values  []*Endpoint
	length  uint64
	pos     uint64
}

func (endpoints *VersionEndpoints) Add(ep *Endpoint) {
	endpoints.values = append(endpoints.values, ep)
	endpoints.length++
	return
}

func (endpoints *VersionEndpoints) Remove(id []byte) (ok bool) {
	if endpoints.length == 0 {
		return
	}
	idStr := bytex.ToString(id)
	pos := -1
	for i, value := range endpoints.values {
		if value.Id() == idStr {
			pos = i
			break
		}
	}
	if pos == -1 {
		return
	}
	endpoints.values = append(endpoints.values[:pos], endpoints.values[pos+1:]...)
	endpoints.length--
	ok = true
	return
}

func (endpoints *VersionEndpoints) Get(id []byte) (ep *Endpoint) {
	if endpoints.length == 0 {
		return
	}
	idStr := bytex.ToString(id)
	for _, value := range endpoints.values {
		if value.Id() == idStr {
			ep = value
			return
		}
	}
	return
}

func (endpoints *VersionEndpoints) Next() (ep *Endpoint) {
	if endpoints.length == 0 {
		return
	}
	for i := uint64(0); i < endpoints.length; i++ {
		pos := atomic.AddUint64(&endpoints.pos, 1) % endpoints.length
		target := endpoints.values[pos]
		if target.Running() && target.IsHealth() {
			ep = target
			return
		}
	}
	return
}

type SortedVersionEndpoints []*VersionEndpoints

func (list SortedVersionEndpoints) Len() int {
	return len(list)
}

func (list SortedVersionEndpoints) Less(i, j int) bool {
	return list[i].version.LessThan(list[j].version)
}

func (list SortedVersionEndpoints) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
	return
}

func (list SortedVersionEndpoints) Get(version versions.Version) *VersionEndpoints {
	for _, vps := range list {
		if vps.version.Equals(version) {
			return vps
		}
	}
	return nil
}

func (list SortedVersionEndpoints) Add(ep *Endpoint) SortedVersionEndpoints {
	vps := list.Get(ep.info.Version)
	if vps == nil {
		vps = &VersionEndpoints{
			version: ep.info.Version,
			values:  make([]*Endpoint, 0),
			length:  0,
			pos:     0,
		}
		vps.Add(ep)
		newList := append(list, vps)
		sort.Sort(newList)
		return newList
	} else {
		vps.Add(ep)
		return list
	}
}

type Endpoints struct {
	values SortedVersionEndpoints
	length uint64
	lock   sync.RWMutex
}

func (endpoints *Endpoints) Add(ep *Endpoint) {
	endpoints.lock.Lock()
	defer endpoints.lock.Unlock()
	endpoints.values = endpoints.values.Add(ep)
	endpoints.length++
	return
}

func (endpoints *Endpoints) Remove(id []byte) {
	endpoints.lock.Lock()
	defer endpoints.lock.Unlock()
	evict := -1
	for i, value := range endpoints.values {
		if value.Remove(id) {
			if value.length == 0 {
				evict = i
			}
			break
		}
	}
	if evict == -1 {
		return
	}
	endpoints.values = append(endpoints.values[:evict], endpoints.values[evict+1:]...)
	endpoints.length--
}

func (endpoints *Endpoints) MaxOne() (ep *Endpoint) {
	endpoints.lock.RLock()
	defer endpoints.lock.RUnlock()
	if endpoints.length == 0 {
		return
	}
	ep = endpoints.values[endpoints.length-1].Next()
	return
}

func (endpoints *Endpoints) Get(id []byte) *Endpoint {
	endpoints.lock.RLock()
	defer endpoints.lock.RUnlock()
	if endpoints.length == 0 {
		return nil
	}
	for _, value := range endpoints.values {
		target := value.Get(id)
		if target == nil {
			continue
		}
		if target.Running() && target.IsHealth() {
			return target
		}
		return nil
	}
	return nil
}

func (endpoints *Endpoints) Range(interval versions.Interval) *Endpoint {
	endpoints.lock.RLock()
	defer endpoints.lock.RUnlock()
	if endpoints.length == 0 {
		return nil
	}
	targets := make([]*Endpoint, 0, 1)
	for _, value := range endpoints.values {
		if interval.Accept(value.version) {
			for _, endpoint := range value.values {
				if endpoint.Running() && endpoint.IsHealth() {
					targets = append(targets, endpoint)
				}
			}
		}
	}
	pos := rand.Intn(len(targets))
	return targets[pos]
}

func (endpoints *Endpoints) Infos() (v services.EndpointInfos) {
	endpoints.lock.RLock()
	defer endpoints.lock.RUnlock()
	if endpoints.length == 0 {
		return
	}
	for _, value := range endpoints.values {
		for _, ep := range value.values {
			v = append(v, ep.info)
		}
	}
	return
}

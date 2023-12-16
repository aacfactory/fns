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
	"sort"
	"sync/atomic"
	"time"
)

func NewEndpoint(address string, id string, version versions.Version, name string, internal bool, document documents.Endpoint, client transports.Client, signature signatures.Signature) (endpoint *Endpoint) {
	endpoint = &Endpoint{
		address:   address,
		id:        id,
		version:   version,
		name:      name,
		internal:  internal,
		document:  document,
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
	address   string
	id        string
	version   versions.Version
	name      string
	internal  bool
	document  documents.Endpoint
	running   atomic.Bool
	functions services.Fns
	client    transports.Client
	signature signatures.Signature
	errs      *window.Times
}

func (endpoint *Endpoint) Running() bool {
	return endpoint.running.Load()
}

func (endpoint *Endpoint) Address() string {
	return endpoint.address
}

func (endpoint *Endpoint) Name() string {
	return endpoint.name
}

func (endpoint *Endpoint) Internal() bool {
	return endpoint.internal
}

func (endpoint *Endpoint) Document() documents.Endpoint {
	return endpoint.document
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
		endpointName: endpoint.name,
		name:         name,
		internal:     internal,
		readonly:     readonly,
		path:         bytex.FromString(fmt.Sprintf("/%s/%s", endpoint.name, name)),
		signature:    endpoint.signature,
		errs:         endpoint.errs,
		health:       atomic.Bool{},
		client:       endpoint.client,
	}
	fn.health.Store(true)
	endpoint.functions = endpoint.functions.Add(fn)
}

func (endpoint *Endpoint) Info() services.EndpointInfo {
	fnInfos := make(services.FnInfos, 0, len(endpoint.functions))
	for _, fn := range endpoint.functions {
		fnInfos = append(fnInfos, services.FnInfo{
			Name:     fn.Name(),
			Readonly: fn.Readonly(),
			Internal: fn.Internal(),
		})
	}
	sort.Sort(fnInfos)
	return services.EndpointInfo{
		Id:        endpoint.id,
		Version:   endpoint.version,
		Address:   endpoint.address,
		Name:      endpoint.name,
		Internal:  endpoint.internal,
		Functions: fnInfos,
		Document:  endpoint.document,
	}
}

type Endpoints []*Endpoint

func (list Endpoints) Len() int {
	return len(list)
}

func (list Endpoints) Less(i, j int) bool {
	return list[i].version.LessThan(list[j].version)
}

func (list Endpoints) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
	return
}

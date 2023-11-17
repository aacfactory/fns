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
	"sync/atomic"
	"time"
)

func NewEndpoint(id string, version versions.Version, name string, internal bool, document documents.Endpoint, client transports.Client, signature signatures.Signature) (endpoint *Endpoint) {
	endpoint = &Endpoint{
		id:        id,
		version:   version,
		name:      name,
		internal:  internal,
		document:  document,
		running:   atomic.Bool{},
		functions: make(services.Fns, 0, 1),
		client:    client,
		signature: signature,
	}
	endpoint.running.Store(true)
	return
}

type Endpoint struct {
	id        string
	version   versions.Version
	name      string
	internal  bool
	document  documents.Endpoint
	running   atomic.Bool
	functions services.Fns
	client    transports.Client
	signature signatures.Signature
}

func (endpoint *Endpoint) Running() bool {
	return endpoint.running.Load()
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

func (endpoint *Endpoint) AddFn(name string, internal bool, readonly bool) {
	fn := &Fn{
		endpointName: endpoint.name,
		name:         name,
		internal:     internal,
		readonly:     readonly,
		path:         bytex.FromString(fmt.Sprintf("/%s/%s", endpoint.name, name)),
		signature:    endpoint.signature,
		errs:         window.NewTimes(10 * time.Second),
		health:       atomic.Bool{},
		client:       endpoint.client,
	}
	fn.health.Store(true)
	endpoint.functions = endpoint.functions.Add(fn)
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

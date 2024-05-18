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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/workers"
	"reflect"
	"sort"
	"sync"
	"time"
)

func NewManager(id string, version versions.Version, address string, cluster Cluster, local services.EndpointsManager, worker workers.Workers, log logs.Logger, dialer transports.Dialer, signature signatures.Signature) ClusterEndpointsManager {
	v := &Manager{
		id:        id,
		version:   version,
		address:   address,
		log:       log.With("cluster", "endpoints"),
		cluster:   cluster,
		local:     local,
		worker:    worker,
		dialer:    dialer,
		signature: signature,
		registration: &Registration{
			values: sync.Map{},
		},
	}
	return v
}

type ClusterEndpointsManager interface {
	services.EndpointsManager
	FnAddress(ctx context.Context, endpoint []byte, fnName []byte, options ...services.EndpointGetOption) (address string, internal bool, has bool)
}

type Manager struct {
	id           string
	version      versions.Version
	address      string
	log          logs.Logger
	cluster      Cluster
	local        services.EndpointsManager
	worker       workers.Workers
	dialer       transports.Dialer
	signature    signatures.Signature
	registration *Registration
}

func (manager *Manager) Add(service services.Service) (err error) {
	err = manager.local.Add(service)
	if err != nil {
		return
	}
	functions := make(services.FnInfos, 0, len(service.Functions()))
	for _, fn := range service.Functions() {
		functions = append(functions, services.FnInfo{
			Name:     fn.Name(),
			Readonly: fn.Readonly(),
			Internal: service.Internal() || fn.Internal(),
		})
	}
	sort.Sort(functions)
	info, infoErr := NewService(service.Name(), service.Internal(), functions, service.Document())
	if infoErr != nil {
		err = errors.Warning("fns: create cluster service info failed").WithCause(infoErr).WithMeta("endpoint", service.Name())
		return
	}
	manager.cluster.AddService(info)
	return
}

func (manager *Manager) Info() (infos services.EndpointInfos) {
	infos = manager.registration.Infos()
	if infos == nil {
		infos = make(services.EndpointInfos, 0)
	}
	local := manager.local.Info()
	infos = append(infos, local...)
	sort.Sort(infos)
	return
}

func (manager *Manager) FnAddress(ctx context.Context, endpoint []byte, fnName []byte, options ...services.EndpointGetOption) (address string, internal bool, has bool) {
	local, inLocal := manager.local.Get(ctx, endpoint, options...)
	if inLocal {
		fnNameString := bytex.ToString(fnName)
		for _, fn := range local.Functions() {
			if fn.Name() == fnNameString {
				address = manager.address
				internal = fn.Internal()
				has = true
				return
			}
		}
	}

	if len(options) == 0 {
		matched := manager.registration.MaxOne(endpoint)
		if matched == nil || reflect.ValueOf(matched).IsNil() {
			return
		}
		if fn, hasFn := matched.Functions().Find(fnName); hasFn {
			address = matched.Address()
			internal = fn.Internal()
			has = true
		}
		return
	}

	opt := services.EndpointGetOptions{}
	for _, option := range options {
		option(&opt)
	}

	interval, hasVersion := opt.Versions().Get(endpoint)
	if hasVersion {
		matched := manager.registration.Range(endpoint, interval)
		if matched == nil || reflect.ValueOf(matched).IsNil() {
			return
		}
		if fn, hasFn := matched.Functions().Find(fnName); hasFn {
			address = matched.Address()
			internal = fn.Internal()
			has = true
		}
		return
	}
	if endpointId := opt.Id(); len(endpointId) > 0 {
		matched := manager.registration.Get(endpoint, endpointId)
		if matched == nil || reflect.ValueOf(matched).IsNil() {
			return
		}
		if fn, hasFn := matched.Functions().Find(fnName); hasFn {
			address = matched.Address()
			internal = fn.Internal()
			has = true
		}
		return
	}
	return
}

func (manager *Manager) Get(ctx context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	// local
	local, inLocal := manager.local.Get(ctx, name, options...)
	if inLocal {
		endpoint = local
		has = true
		return
	}
	// max one
	if len(options) == 0 {
		endpoint = manager.registration.MaxOne(name)
		has = !reflect.ValueOf(endpoint).IsNil()
		return
	}

	opt := services.EndpointGetOptions{}
	for _, option := range options {
		option(&opt)
	}
	// get by id
	if eid := opt.Id(); len(eid) > 0 {
		endpoint = manager.registration.Get(name, eid)
		has = !reflect.ValueOf(endpoint).IsNil()
		return
	}
	// get by intervals
	if intervals := opt.Versions(); len(intervals) > 0 {
		interval, matched := intervals.Get(name)
		if !matched {
			return
		}
		endpoint = manager.registration.Range(name, interval)
		has = !reflect.ValueOf(endpoint).IsNil()
		return
	}
	return
}

func (manager *Manager) RequestAsync(ctx context.Context, name []byte, fn []byte, param any, options ...services.RequestOption) (future futures.Future, err error) {
	// valid params
	if len(name) == 0 {
		err = errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("name is nil"))
		return
	}
	if len(fn) == 0 {
		err = errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("fn is nil"))
		return
	}
	// request
	req := services.NewRequest(ctx, name, fn, param, options...)
	// get endpoint
	var endpointGetOptions []services.EndpointGetOption
	if endpointId := req.Header().EndpointId(); len(endpointId) > 0 {
		endpointGetOptions = make([]services.EndpointGetOption, 0, 1)
		endpointGetOptions = append(endpointGetOptions, services.EndpointId(endpointId))
	}
	if acceptedVersions := req.Header().AcceptedVersions(); len(acceptedVersions) > 0 {
		if endpointGetOptions == nil {
			endpointGetOptions = make([]services.EndpointGetOption, 0, 1)
		}
		endpointGetOptions = append(endpointGetOptions, services.EndpointVersions(acceptedVersions))
	}
	endpoint, found := manager.Get(req, name, endpointGetOptions...)
	if !found {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("endpoint", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		return
	}
	// get fn
	function, hasFunction := endpoint.Functions().Find(fn)
	if !hasFunction {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("endpoint", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		return
	}
	// log
	logs.With(req, manager.log.With("service", bytex.ToString(name)).With("fn", bytex.ToString(fn)))
	// components
	service, ok := endpoint.(services.Service)
	if ok {
		components := service.Components()
		if len(components) > 0 {
			services.WithComponents(req, name, components)
		}
	}
	// tracing
	trace, hasTrace := tracings.Load(req)
	if hasTrace {
		trace.Begin(req.Header().ProcessId(), name, fn, "scope", "local")
	}
	// promise
	var promise futures.Promise
	promise, future = futures.New()
	// dispatch
	dispatched := manager.worker.Dispatch(req, services.FnTask{
		Fn:      function,
		Promise: promise,
	})
	if !dispatched {
		// release promise
		futures.ReleaseUnused(promise)
		future = nil
		// tracing
		if hasTrace {
			trace.Finish("succeed", "false", "cause", "***TOO MANY REQUEST***")
		}
		err = errors.TooMayRequest("fns: too may request, try again later.").
			WithMeta("endpoint", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
	}
	return
}

func (manager *Manager) Request(ctx context.Context, name []byte, fn []byte, param interface{}, options ...services.RequestOption) (response services.Response, err error) {
	// valid params
	if len(name) == 0 {
		err = errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("name is nil"))
		return
	}
	if len(fn) == 0 {
		err = errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("fn is nil"))
		return
	}

	// request
	req := services.AcquireRequest(ctx, name, fn, param, options...)
	defer services.ReleaseRequest(req)
	// get endpoint
	var endpointGetOptions []services.EndpointGetOption
	if endpointId := req.Header().EndpointId(); len(endpointId) > 0 {
		endpointGetOptions = make([]services.EndpointGetOption, 0, 1)
		endpointGetOptions = append(endpointGetOptions, services.EndpointId(endpointId))
	}
	if acceptedVersions := req.Header().AcceptedVersions(); len(acceptedVersions) > 0 {
		if endpointGetOptions == nil {
			endpointGetOptions = make([]services.EndpointGetOption, 0, 1)
		}
		endpointGetOptions = append(endpointGetOptions, services.EndpointVersions(acceptedVersions))
	}
	endpoint, found := manager.Get(req, name, endpointGetOptions...)
	if !found {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("endpoint", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		return
	}
	// get fn
	function, hasFunction := endpoint.Functions().Find(fn)
	if !hasFunction {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("endpoint", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		return
	}
	// log
	logs.With(req, manager.log.With("service", bytex.ToString(name)).With("fn", bytex.ToString(fn)))
	// components
	service, ok := endpoint.(services.Service)
	if ok {
		components := service.Components()
		if len(components) > 0 {
			services.WithComponents(req, name, components)
		}
	}
	// tracing
	trace, hasTrace := tracings.Load(req)
	if hasTrace {
		trace.Begin(req.Header().ProcessId(), name, fn, "scope", "local")
		trace.Waited()
	}
	// handle
	result, handleErr := function.Handle(req)
	if handleErr != nil {
		codeErr := errors.Wrap(handleErr).WithMeta("endpoint", bytex.ToString(name)).WithMeta("fn", bytex.ToString(fn))
		if hasTrace {
			trace.Finish("succeed", "false", "cause", codeErr.Name())
		}
		err = codeErr
		return
	}
	if hasTrace {
		trace.Finish("succeed", "true")
	}
	response = services.NewResponse(result)
	return
}

func (manager *Manager) Listen(ctx context.Context) (err error) {
	// watching
	manager.watching()
	// cluster.join
	err = manager.cluster.Join(ctx)
	if err != nil {
		if manager.log.WarnEnabled() {
			manager.log.Warn().With("cluster", "join").Cause(err).Message("fns: cluster join failed")
		}
		return
	}
	// local.listen
	err = manager.local.Listen(ctx)
	if err != nil {
		_ = manager.cluster.Leave(ctx)
		return
	}
	return
}

func (manager *Manager) Shutdown(ctx context.Context) {
	leaveErr := manager.cluster.Leave(ctx)
	if leaveErr != nil {
		if manager.log.WarnEnabled() {
			manager.log.Warn().With("cluster", "leave").Cause(leaveErr).Message("fns: cluster leave failed")
		}
	}
	manager.local.Shutdown(ctx)
	return
}

func (manager *Manager) watching() {
	go func(eps *Manager) {
		for {
			event, ok := <-eps.cluster.NodeEvents()
			if !ok {
				break
			}
			if eps.log.DebugEnabled() {
				eps.log.Debug().
					With("event", event.Kind.String()).
					Message(fmt.Sprintf(
						"fns: get node(id:%s addr:%s ver:%s, services:%d) event(%s)",
						event.Node.Id, event.Node.Address, event.Node.Version.String(), len(event.Node.Services), event.Kind.String()))
			}
			switch event.Kind {
			case Add:
				endpoints := make([]*Endpoint, 0, 1)
				client, clientErr := eps.dialer.Dial(bytex.FromString(event.Node.Address))
				if eps.log.DebugEnabled() {
					succeed := "succeed"
					var cause error
					if clientErr != nil {
						succeed = "failed"
						cause = errors.Warning(fmt.Sprintf("fns: dial %s failed", event.Node.Address)).WithMeta("address", event.Node.Address).WithCause(clientErr)
					}
					eps.log.Debug().
						With("cluster", "registrations").
						Cause(cause).
						Message(fmt.Sprintf("fns: dial %s %s", event.Node.Address, succeed))
				}
				if clientErr != nil {
					if eps.log.WarnEnabled() {
						eps.log.Warn().
							With("cluster", "registrations").
							Cause(errors.Warning(fmt.Sprintf("fns: dial %s failed", event.Node.Address)).WithMeta("address", event.Node.Address).WithCause(clientErr)).
							Message(fmt.Sprintf("fns: dial %s failed", event.Node.Address))
					}
					break
				}
				// check health
				active := false
				for i := 0; i < 10; i++ {
					ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
					if runtime.CheckHealth(ctx, client) {
						active = true
						cancel()
						break
					}
					cancel()
					if eps.log.DebugEnabled() {
						eps.log.Debug().With("cluster", "registrations").Message(fmt.Sprintf("fns: %s is not health", event.Node.Address))
					}
					time.Sleep(1 * time.Second)
				}

				if eps.log.DebugEnabled() {
					eps.log.Debug().With("cluster", "registrations").Message(fmt.Sprintf("fns: health of %s is %v", event.Node.Address, active))
				}
				if !active {
					break
				}
				// get document
				for _, endpoint := range event.Node.Services {
					document, documentErr := endpoint.Document()
					if documentErr != nil {
						if eps.log.WarnEnabled() {
							eps.log.Warn().
								With("cluster", "registrations").
								Cause(errors.Warning("fns: get endpoint document failed").WithMeta("address", event.Node.Address).WithCause(documentErr)).
								Message(fmt.Sprintf("fns: dial %s failed", event.Node.Address))
						}
						continue
					}
					ep := NewEndpoint(manager.log, event.Node.Address, event.Node.Id, event.Node.Version, endpoint.Name, endpoint.Internal, document, client, eps.signature)
					for _, fnInfo := range endpoint.Functions {
						ep.AddFn(fnInfo.Name, fnInfo.Internal, fnInfo.Readonly)
					}
					endpoints = append(endpoints, ep)
				}
				for _, endpoint := range endpoints {
					eps.registration.Add(endpoint)
				}
				if eps.log.DebugEnabled() {
					eps.log.Debug().With("cluster", "registrations").Message(fmt.Sprintf("fns: %s added", event.Node.Address))
				}
				break
			case Remove:
				for _, endpoint := range event.Node.Services {
					eps.registration.Remove(endpoint.Name, event.Node.Id)
				}
				break
			default:
				break
			}
		}
	}(manager)
	return
}

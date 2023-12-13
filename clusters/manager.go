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
	fLog "github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"net/http"
	"sort"
	"sync"
	"time"
)

func NewManager(id string, version versions.Version, address string, cluster Cluster, local services.EndpointsManager, worker workers.Workers, log logs.Logger, dialer transports.Dialer, signature signatures.Signature) ClusterEndpointsManager {
	v := &Manager{
		id:            id,
		version:       version,
		address:       address,
		log:           log.With("cluster", "endpoints"),
		cluster:       cluster,
		local:         local,
		worker:        worker,
		dialer:        dialer,
		signature:     signature,
		registrations: make(Registrations, 0, 1),
		infos:         nil,
		locker:        sync.RWMutex{},
	}
	return v
}

type ClusterEndpointsManager interface {
	services.EndpointsManager
	Address() string
	PublicFnAddress(ctx context.Context, endpoint []byte, fnName []byte, options ...services.EndpointGetOption) (address string, has bool)
}

type Manager struct {
	id            string
	version       versions.Version
	address       string
	log           logs.Logger
	cluster       Cluster
	local         services.EndpointsManager
	worker        workers.Workers
	dialer        transports.Dialer
	signature     signatures.Signature
	registrations Registrations
	infos         services.EndpointInfos
	locker        sync.RWMutex
}

func (manager *Manager) Address() string {
	return manager.address
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
	manager.locker.RLock()
	infos = manager.infos
	manager.locker.RUnlock()
	return
}

func (manager *Manager) PublicFnAddress(ctx context.Context, endpoint []byte, fnName []byte, options ...services.EndpointGetOption) (address string, has bool) {
	local, localed := manager.local.Get(ctx, endpoint, options...)
	if localed {
		if local.Internal() {
			return
		}
		fnNameString := bytex.ToString(fnName)
		for _, fn := range local.Functions() {
			if fn.Name() == fnNameString {
				if fn.Internal() {
					return
				}
				has = true
				return
			}
		}
	}
	manager.locker.RLock()
	registration, found := manager.registrations.Get(endpoint)
	if !found {
		manager.locker.RUnlock()
		return
	}
	if len(options) == 0 {
		maxed, exist := registration.MaxOne()
		if !exist {
			manager.locker.RUnlock()
			return
		}
		address = maxed.Address()
		has = true
		manager.locker.RUnlock()
		return
	}
	opt := services.EndpointGetOptions{}
	for _, option := range options {
		option(&opt)
	}

	interval, hasVersion := opt.Versions().Get(endpoint)
	if hasVersion {
		matched, exist := registration.Range(interval)
		if !exist {
			manager.locker.RUnlock()
			return
		}
		address = matched.Address()
		has = true
		manager.locker.RUnlock()
		return
	}
	manager.locker.RUnlock()
	return
}

func (manager *Manager) Get(ctx context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	local, localed := manager.local.Get(ctx, name, options...)
	if localed {
		endpoint = local
		has = true
		return
	}
	manager.locker.RLock()
	registration, found := manager.registrations.Get(name)
	if !found {
		manager.locker.RUnlock()
		return
	}
	if len(options) == 0 {
		endpoint, has = registration.MaxOne()
		manager.locker.RUnlock()
		return
	}
	opt := services.EndpointGetOptions{}
	for _, option := range options {
		option(&opt)
	}
	if eid := opt.Id(); len(eid) > 0 {
		endpoint, has = registration.Get(eid)
		manager.locker.RUnlock()
		return
	}
	if intervals := opt.Versions(); len(intervals) > 0 {
		interval, matched := intervals.Get(name)
		if !matched {
			manager.locker.RUnlock()
			return
		}
		endpoint, has = registration.Range(interval)
		manager.locker.RUnlock()
		return
	}
	manager.locker.RUnlock()
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
	endpoint, found := manager.Get(ctx, name, endpointGetOptions...)
	if !found {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("endpoint", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		services.ReleaseRequest(req)
		return
	}

	function, hasFunction := endpoint.Functions().Find(fn)
	if !hasFunction {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("endpoint", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		services.ReleaseRequest(req)
		return
	}

	// ctx >>>
	ctx = req
	// log
	fLog.With(ctx, manager.log.With("service", bytex.ToString(name)).With("fn", bytex.ToString(fn)))
	// components
	service, ok := endpoint.(services.Service)
	if ok {
		components := service.Components()
		if len(components) > 0 {
			services.WithComponents(ctx, name, components)
		}
	}
	// ctx <<<

	// tracing
	trace, hasTrace := tracings.Load(ctx)
	if hasTrace {
		trace.Begin(req.Header().ProcessId(), name, fn, "scope", "local")
	}
	// promise
	promise, future := futures.New()
	// dispatch
	dispatched := manager.worker.Dispatch(ctx, services.FnTask{
		Fn:      function,
		Promise: promise,
	})
	if !dispatched {
		// tracing
		if hasTrace {
			trace.Finish("succeed", "false", "cause", "***TOO MANY REQUEST***")
		}
		promise.Failed(
			errors.New(http.StatusTooManyRequests, "***TOO MANY REQUEST***", "fns: too may request, try again later.").
				WithMeta("endpoint", bytex.ToString(name)).
				WithMeta("fn", bytex.ToString(fn)),
		)
	}
	response, err = future.Get(ctx)
	services.ReleaseRequest(req)
	return
}

func (manager *Manager) Listen(ctx context.Context) (err error) {
	// copy info
	manager.infos = manager.local.Info()
	for i, info := range manager.infos {
		info.Address = manager.address
		manager.infos[i] = info
	}
	// watching
	manager.watching()
	// local.listen
	err = manager.local.Listen(ctx)
	if err != nil {
		return
	}
	// cluster.join
	err = manager.cluster.Join(ctx)
	if err != nil {
		if manager.log.WarnEnabled() {
			manager.log.Warn().With("cluster", "join").Cause(err).Message("fns: cluster join failed")
		}
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
						"fns: get node(id:%s addr:%s ver:%s, services:%d) event",
						event.Node.Id, event.Node.Address, event.Node.Version.String(), len(event.Node.Services)))
			}
			switch event.Kind {
			case Add:
				endpoints := make(Endpoints, 0, 1)
				client, clientErr := eps.dialer.Dial(bytex.FromString(event.Node.Address))
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
					time.Sleep(1 * time.Second)
				}
				if !active {
					break
				}
				// get document
				infos := eps.local.Info()[:]
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
					ep := NewEndpoint(event.Node.Address, event.Node.Id, event.Node.Version, endpoint.Name, endpoint.Internal, document, client, eps.signature)
					for _, fnInfo := range endpoint.Functions {
						ep.AddFn(fnInfo.Name, fnInfo.Internal, fnInfo.Readonly)
					}
					endpoints = append(endpoints, ep)
					infos = append(infos, ep.Info())
				}
				sort.Sort(infos)
				eps.locker.Lock()
				for _, endpoint := range endpoints {
					eps.registrations = eps.registrations.Add(endpoint)
				}
				eps.infos = infos
				eps.locker.Unlock()
				break
			case Remove:
				eps.locker.Lock()
				for _, endpoint := range event.Node.Services {
					eps.registrations = eps.registrations.Remove(endpoint.Name, event.Node.Id)
					idx := -1
					for i, info := range eps.infos {
						if info.Id == event.Node.Id && info.Name == endpoint.Name {
							idx = i
							break
						}
					}
					if idx != -1 {
						eps.infos = append(eps.infos[:idx], eps.infos[idx+1:]...)
					}
				}
				eps.locker.Unlock()
				break
			default:
				break
			}
		}
	}(manager)
	return
}

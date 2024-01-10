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

package services

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	fLog "github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"sort"
	"strings"
	"sync"
	"time"
)

func New(id string, version versions.Version, log logs.Logger, config Config, worker workers.Workers) EndpointsManager {
	return &Manager{
		log:     log,
		config:  config,
		id:      id,
		version: version,
		values:  make(Services, 0, 1),
		infos:   make(EndpointInfos, 0, 1),
		worker:  worker,
	}
}

type EndpointsManager interface {
	Endpoints
	Add(service Service) (err error)
	Listen(ctx context.Context) (err error)
	Shutdown(ctx context.Context)
}

type Manager struct {
	log     logs.Logger
	config  Config
	id      string
	version versions.Version
	values  Services
	infos   EndpointInfos
	worker  workers.Workers
}

func (manager *Manager) Add(service Service) (err error) {
	name := strings.TrimSpace(service.Name())
	if _, has := manager.values.Find([]byte(name)); has {
		err = errors.Warning("fns: services add service failed").WithMeta("service", name).WithCause(fmt.Errorf("service has added"))
		return
	}
	config, configErr := manager.config.Get(name)
	if configErr != nil {
		err = errors.Warning("fns: services add service failed").WithMeta("service", name).WithCause(configErr)
		return
	}
	constructErr := service.Construct(Options{
		Id:      manager.id,
		Version: manager.version,
		Log:     manager.log.With("service", name),
		Config:  config,
	})
	if constructErr != nil {
		err = errors.Warning("fns: services add service failed").WithMeta("service", name).WithCause(constructErr)
		return
	}
	manager.values = manager.values.Add(service)
	// info
	internal := service.Internal()
	functions := make(FnInfos, 0, len(service.Functions()))
	for _, fn := range service.Functions() {
		functions = append(functions, FnInfo{
			Name:     fn.Name(),
			Readonly: fn.Readonly(),
			Internal: internal || fn.Internal(),
		})
	}
	sort.Sort(functions)
	manager.infos = append(manager.infos, EndpointInfo{
		Id:        manager.id,
		Name:      service.Name(),
		Version:   manager.version,
		Internal:  internal,
		Functions: functions,
		Document:  service.Document(),
	})
	sort.Sort(manager.infos)
	return
}

func (manager *Manager) Info() (infos EndpointInfos) {
	infos = manager.infos
	return
}

func (manager *Manager) Get(_ context.Context, name []byte, options ...EndpointGetOption) (endpoint Endpoint, has bool) {
	if len(options) > 0 {
		opt := EndpointGetOptions{
			id:              nil,
			requestVersions: nil,
		}
		for _, option := range options {
			option(&opt)
		}
		if len(opt.id) > 0 {
			if manager.id != string(opt.id) {
				return
			}
		}
		if len(opt.requestVersions) > 0 {
			if !opt.requestVersions.Accept(name, manager.version) {
				return
			}
		}
	}
	endpoint, has = manager.values.Find(name)
	return
}

func (manager *Manager) RequestAsync(req Request) (future futures.Future, err error) {

	var endpointGetOptions []EndpointGetOption
	if endpointId := req.Header().EndpointId(); len(endpointId) > 0 {
		endpointGetOptions = make([]EndpointGetOption, 0, 1)
		endpointGetOptions = append(endpointGetOptions, EndpointId(endpointId))
	}
	if acceptedVersions := req.Header().AcceptedVersions(); len(acceptedVersions) > 0 {
		if endpointGetOptions == nil {
			endpointGetOptions = make([]EndpointGetOption, 0, 1)
		}
		endpointGetOptions = append(endpointGetOptions, EndpointVersions(acceptedVersions))
	}

	name, fn := req.Fn()

	endpoint, found := manager.Get(req, name, endpointGetOptions...)
	if !found {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("endpoint", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		return
	}

	function, hasFunction := endpoint.Functions().Find(fn)
	if !hasFunction {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("endpoint", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		return
	}

	// log
	fLog.With(req, manager.log.With("service", bytex.ToString(name)).With("fn", bytex.ToString(fn)))
	// components
	service, ok := endpoint.(Service)
	if ok {
		components := service.Components()
		if len(components) > 0 {
			WithComponents(req, name, components)
		}
	}
	// ctx <<<
	// tracing
	trace, hasTrace := tracings.Load(req)
	if hasTrace {
		trace.Begin(req.Header().ProcessId(), name, fn, "scope", "local")
	}

	// promise
	var promise futures.Promise
	promise, future = futures.New()
	// dispatch
	dispatched := manager.worker.Dispatch(req, FnTask{
		Fn:      function,
		Promise: promise,
	})
	if !dispatched {
		// release futures
		futures.Release(future)
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

func (manager *Manager) Request(ctx context.Context, name []byte, fn []byte, param interface{}, options ...RequestOption) (response Response, err error) {
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
	req := AcquireRequest(ctx, name, fn, param, options...)
	defer ReleaseRequest(req)

	future, reqErr := manager.RequestAsync(req)
	if reqErr != nil {
		err = reqErr
		return
	}
	response, err = future.Get(ctx)
	return
}

func (manager *Manager) Listen(ctx context.Context) (err error) {
	errs := errors.MakeErrors()
	for _, endpoint := range manager.values {
		ln, ok := endpoint.(Listenable)
		if !ok {
			continue
		}
		errCh := make(chan error, 1)
		name := ln.Name()
		lnCtx := context.WithValue(ctx, "listener", name)
		fLog.With(lnCtx, manager.log.With("service", name))
		if components := ln.Components(); len(components) > 0 {
			WithComponents(lnCtx, bytex.FromString(name), components)
		}
		go func(ctx context.Context, ln Listenable, errCh chan error) {
			lnErr := ln.Listen(ctx)
			if lnErr != nil {
				errCh <- lnErr
			}
		}(lnCtx, ln, errCh)
		select {
		case lnErr := <-errCh:
			errs.Append(lnErr)
			break
		case <-time.After(5 * time.Second):
			break
		}
		close(errCh)
		if manager.log.DebugEnabled() {
			manager.log.Debug().With("service", endpoint.Name()).Message("fns: service is listening...")
		}
	}
	if len(errs) > 0 {
		err = errs.Error()
	}
	return
}

func (manager *Manager) Shutdown(ctx context.Context) {
	wg := new(sync.WaitGroup)
	ch := make(chan struct{}, 1)
	for _, endpoint := range manager.values {
		wg.Add(1)
		go func(ctx context.Context, endpoint Endpoint, wg *sync.WaitGroup) {
			endpoint.Shutdown(ctx)
			wg.Done()
		}(ctx, endpoint, wg)
	}
	go func(ch chan struct{}, wg *sync.WaitGroup) {
		wg.Wait()
		ch <- struct{}{}
		close(ch)
	}(ch, wg)
	select {
	case <-ctx.Done():
		break
	case <-ch:
		break
	}
}

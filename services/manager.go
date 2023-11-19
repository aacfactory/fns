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

package services

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/commons/mmhash"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/log"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"golang.org/x/sync/singleflight"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

func New(id string, version versions.Version, log logs.Logger, config Config, worker workers.Workers) EndpointsManager {
	return &Manager{
		log:     log.With("fns", "services"),
		config:  config,
		id:      id,
		version: version,
		values:  make(Services, 0, 1),
		infos:   make(EndpointInfos, 0, 1),
		worker:  worker,
		group:   new(singleflight.Group),
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
	group   *singleflight.Group
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
	endpoint, found := manager.Get(ctx, name, endpointGetOptions...)
	if !found {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("service", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		ReleaseRequest(req)
		return
	}

	function, hasFunction := endpoint.Functions().Find(fn)
	if !hasFunction {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("service", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		ReleaseRequest(req)
		return
	}

	// ctx >>>
	ctx = req
	// log
	log.With(ctx, manager.log.With("service", bytex.ToString(name)).With("fn", bytex.ToString(fn)))
	// components
	service, ok := endpoint.(Service)
	if ok {
		components := service.Components()
		if len(components) > 0 {
			WithComponents(ctx, components)
		}
	}
	// ctx <<<
	// tracing
	trace, hasTrace := tracings.Load(ctx)
	if hasTrace {
		trace.Begin(req.Header().ProcessId(), name, fn, "scope", "local")
	}

	groupKey, groupKeyErr := HashRequest(req, HashRequestWithDeviceId(), HashRequestWithToken(), HashRequestBySumFn(mmhash.Sum64))
	if groupKeyErr != nil {
		err = errors.Warning("fns: hash request failed").
			WithCause(groupKeyErr).
			WithMeta("service", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		ReleaseRequest(req)
		return
	}

	v, doErr, _ := manager.group.Do(bytex.ToString(groupKey), func() (v interface{}, err error) {
		// promise
		promise, future := futures.New()
		// dispatch
		dispatched := manager.worker.Dispatch(ctx, FnTask{
			Fn:      function,
			Promise: promise,
		})
		if !dispatched {
			// release futures
			futures.Release(promise, future)
			// tracing
			if hasTrace {
				trace.Finish("succeed", "false", "cause", "***TOO MANY REQUEST***")
			}
			err = errors.New(http.StatusTooManyRequests, "***TOO MANY REQUEST***", "fns: too may request, try again later.").
				WithMeta("service", bytex.ToString(name)).
				WithMeta("fn", bytex.ToString(fn))
			return
		}
		v, err = future.Get(ctx)
		return
	})
	if doErr != nil {
		err = doErr
	} else {
		response = v.(Response)
	}
	ReleaseRequest(req)
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

		lnCtx := context.WithValue(ctx, bytex.FromString("listener"), ln.Name())
		log.With(lnCtx, manager.log.With("service", ln.Name()))
		if components := ln.Components(); len(components) > 0 {
			WithComponents(lnCtx, components)
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

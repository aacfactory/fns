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

package service

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/internal/configure"
	"github.com/aacfactory/fns/internal/shareds"
	"github.com/aacfactory/fns/shared"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"strings"
	"sync/atomic"
	"time"
)

type Endpoint interface {
	Name() (name string)
	Document() (document Document)
	Request(ctx context.Context, request Request) (result Result)
}

type EndpointDiscoveryGetOption func(options *EndpointDiscoveryGetOptions)

type EndpointDiscoveryGetOptions struct {
	scope string
	id    string
}

func Exact(id string) EndpointDiscoveryGetOption {
	return func(options *EndpointDiscoveryGetOptions) {
		options.id = strings.TrimSpace(id)
		return
	}
}

const (
	localScoped = "local"
)

func LocalScoped() EndpointDiscoveryGetOption {
	return func(options *EndpointDiscoveryGetOptions) {
		options.scope = localScoped
		return
	}
}

type EndpointDiscovery interface {
	Get(ctx context.Context, service string, options ...EndpointDiscoveryGetOption) (endpoint Endpoint, has bool)
	List(ctx context.Context, options ...EndpointDiscoveryGetOption) (endpoints []Endpoint)
}

// +-------------------------------------------------------------------------------------------------------------------+

type EndpointsOptions struct {
	AppId  string
	Log    logs.Logger
	Config *configure.Runtime
}

func NewEndpoints(options EndpointsOptions) (v *Endpoints, err error) {
	// config
	config := options.Config
	if config == nil {
		config = &configure.Runtime{}
	}
	// workers
	maxWorkers := config.MaxWorkers
	if maxWorkers < 1 {
		maxWorkers = 256 * 1024
	}
	maxIdleWorkerSeconds := config.WorkerMaxIdleSeconds
	if maxIdleWorkerSeconds < 1 {
		maxIdleWorkerSeconds = 60
	}
	worker := workers.New(workers.MaxWorkers(maxWorkers), workers.MaxIdleWorkerDuration(time.Duration(maxIdleWorkerSeconds)*time.Second))
	// shared store
	sharedMemSizeStr := strings.TrimSpace(config.LocalSharedMemSize)
	if sharedMemSizeStr == "" {
		sharedMemSizeStr = "64M"
	}
	sharedMemSize, sharedMemSizeErr := commons.ToBytes(sharedMemSizeStr)
	if sharedMemSizeErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(sharedMemSizeErr)
		return
	}
	sharedStore, createSharedStoreErr := shareds.NewLocalStore(int64(sharedMemSize))
	if createSharedStoreErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(createSharedStoreErr)
		return
	}
	// timeout
	handleTimeoutSeconds := options.Config.HandleTimeoutSeconds
	if handleTimeoutSeconds < 1 {
		handleTimeoutSeconds = 10
	}
	v = &Endpoints{
		log:           options.Log,
		appId:         options.AppId,
		running:       commons.NewSafeFlag(false),
		barrier:       DefaultBarrier(),
		worker:        worker,
		sharedLockers: shareds.NewLocalLockers(),
		sharedStore:   sharedStore,
		handleTimeout: time.Duration(handleTimeoutSeconds) * time.Second,
		discovery:     nil,
		deployed:      make(map[string]*endpoint),
	}
	return
}

type Endpoints struct {
	log           logs.Logger
	appId         string
	running       *commons.SafeFlag
	worker        Workers
	handleTimeout time.Duration
	barrier       Barrier
	sharedLockers shared.Lockers
	sharedStore   shared.Store
	discovery     EndpointDiscovery
	deployed      map[string]*endpoint
}

func (e *Endpoints) Run() {
	e.running.On()
}

func (e *Endpoints) IsRunning() (ok bool) {
	ok = e.running.IsOn()
	return
}

func (e *Endpoints) Upgrade(barrier Barrier, store shared.Store, lockers shared.Lockers, discovery EndpointDiscovery) {
	e.barrier = barrier
	if e.sharedStore != nil {
		e.sharedStore.Close()
	}
	e.sharedStore = store
	e.sharedLockers = lockers
	e.discovery = discovery
}

func (e *Endpoints) Get(ctx context.Context, service string, options ...EndpointDiscoveryGetOption) (endpoint Endpoint, has bool) {
	if service == "" {
		return
	}
	opt := &EndpointDiscoveryGetOptions{
		scope: "",
		id:    "",
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	if opt.id != "" {
		if opt.id == e.appId {
			endpoint, has = e.deployed[service]
			return
		}
		if opt.scope != localScoped && e.discovery != nil && CanAccessInternal(ctx) {
			endpoint, has = e.discovery.Get(ctx, service, options...)
		}
	} else {
		endpoint, has = e.deployed[service]
		if !has && opt.scope != localScoped && e.discovery != nil && CanAccessInternal(ctx) {
			endpoint, has = e.discovery.Get(ctx, service)
		}
	}
	return
}

func (e *Endpoints) List(ctx context.Context, options ...EndpointDiscoveryGetOption) (endpoints []Endpoint) {
	opt := &EndpointDiscoveryGetOptions{
		scope: "",
		id:    "",
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	endpoints = make([]Endpoint, 0, 1)
	for _, ep := range e.deployed {
		endpoints = append(endpoints, ep)
	}
	if opt.scope == localScoped {
		return
	}
	if e.discovery == nil || !CanAccessInternal(ctx) {
		return
	}
	remotes := e.discovery.List(ctx)
	if remotes != nil && len(remotes) > 0 {
		endpoints = append(endpoints, remotes...)
	}
	return
}

func (e *Endpoints) Mount(svc Service) {
	e.deployed[svc.Name()] = &endpoint{
		appId:         e.appId,
		log:           e.log,
		discovery:     e.discovery,
		barrier:       e.barrier,
		sharedLockers: e.sharedLockers,
		sharedStore:   e.sharedStore,
		handleTimeout: e.handleTimeout,
		worker:        e.worker,
		svc:           svc,
		running:       e.running,
	}
}

func (e *Endpoints) Listen() (err error) {
	errCh := make(chan error, 8)
	lns := 0
	closed := int64(0)
	for _, ep := range e.deployed {
		ln, ok := ep.svc.(Listenable)
		if !ok {
			continue
		}
		lns++
		ctx := withRuntime(context.TODO(), e.appId, e.log, e.worker, e.discovery, e.barrier, e.sharedLockers, e.sharedStore, e.running)
		go func(ctx context.Context, ln Listenable, errCh chan error) {
			lnErr := ln.Listen(ctx)
			if lnErr != nil {
				lnErr = errors.Warning(fmt.Sprintf("fns: %s listen falied", ln.Name())).WithCause(lnErr).WithMeta("service", ln.Name())
				if atomic.LoadInt64(&closed) == 0 {
					errCh <- lnErr
				}
			}
		}(ctx, ln, errCh)
	}
	if lns == 0 {
		close(errCh)
		return
	}
	select {
	case lnErr := <-errCh:
		atomic.AddInt64(&closed, 1)
		err = errors.Warning("fns: endpoints listen failed").WithCause(lnErr)
		break
	case <-time.After(time.Duration(lns*3) * time.Second):
		break
	}
	close(errCh)
	return
}

func (e *Endpoints) Close() {
	e.running.Off()
	for _, ep := range e.deployed {
		ep.svc.Close()
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

type endpoint struct {
	appId         string
	running       *commons.SafeFlag
	log           logs.Logger
	discovery     EndpointDiscovery
	barrier       Barrier
	sharedLockers shared.Lockers
	sharedStore   shared.Store
	handleTimeout time.Duration
	worker        Workers
	svc           Service
}

func (e *endpoint) Name() (name string) {
	name = e.svc.Name()
	return
}

func (e *endpoint) Document() (document Document) {
	if e.svc.Internal() {
		return
	}
	document = e.svc.Document()
	return
}

func (e *endpoint) Request(ctx context.Context, req Request) (result Result) {
	ctx = withRuntime(ctx, e.appId, e.log, e.worker, e.discovery, e.barrier, e.sharedLockers, e.sharedStore, e.running)
	ctx = withRequest(ctx, req)
	ctx = withTracer(ctx)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, e.handleTimeout)

	fr := NewResult()
	if !e.worker.Dispatch(ctx, newFnTask(e.svc, req, fr)) {
		serviceName, fnName := req.Fn()
		if ctx.Err() != nil {
			fr.Failed(errors.Timeout("fns: workers handle timeout").WithMeta("fns", "timeout").WithMeta("service", serviceName).WithMeta("fn", fnName))
		} else {
			fr.Failed(errors.NotAcceptable("fns: service is overload").WithMeta("fns", "overload").WithMeta("service", serviceName).WithMeta("fn", fnName))
		}
	}
	result = fr

	tryReportTracer(ctx)
	cancel()
	return
}

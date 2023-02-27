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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/commons/flags"
	"github.com/aacfactory/fns/service/shared"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"strings"
	"sync/atomic"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type EndpointsOptions struct {
	AppId      string
	AppVersion versions.Version
	Log        logs.Logger
	Running    *flags.Flag
	Config     *Config
}

func NewEndpoints(options EndpointsOptions) (v *Endpoints, err error) {
	// config

	// log >>>

	// log <<<
	// workers >>>
	config := options.Config
	if config == nil {
		config = &configure.Runtime{}
	}
	maxWorkers := config.MaxWorkers
	if maxWorkers < 1 {
		maxWorkers = 256 * 1024
	}
	maxIdleWorkerSeconds := config.WorkerMaxIdleSeconds
	if maxIdleWorkerSeconds < 1 {
		maxIdleWorkerSeconds = 60
	}
	worker := workers.New(workers.MaxWorkers(maxWorkers), workers.MaxIdleWorkerDuration(time.Duration(maxIdleWorkerSeconds)*time.Second))
	// workers <<<
	// shared store >>>
	sharedMemSizeStr := strings.TrimSpace(config.LocalSharedMemSize)
	if sharedMemSizeStr == "" {
		sharedMemSizeStr = "64M"
	}
	sharedMemSize, sharedMemSizeErr := bytex.ToBytes(sharedMemSizeStr)
	if sharedMemSizeErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(sharedMemSizeErr)
		return
	}
	sharedStore, createSharedStoreErr := shareds.NewLocalStore(int64(sharedMemSize))
	if createSharedStoreErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(createSharedStoreErr)
		return
	}
	// shared store <<<
	// shared lockers >>>
	// shared lockers <<<

	// barrier >>>
	// barrier <<<

	// cluster >>>
	// cluster <<<

	// http >>>
	// http <<<

	// timeout
	handleTimeoutSeconds := options.Config.HandleTimeoutSeconds
	if handleTimeoutSeconds < 1 {
		handleTimeoutSeconds = 10
	}
	v = &Endpoints{
		rt: &runtimes{
			appId:         options.AppId,
			running:       flags.New(false),
			log:           options.Log,
			worker:        worker,
			discovery:     nil,
			barrier:       nil,
			sharedLockers: nil,
			sharedStore:   nil,
		},
		log:           options.Log,
		appId:         options.AppId,
		appVersion:    options.AppVersion,
		running:       options.Running,
		barrier:       defaultBarrier(),
		worker:        worker,
		sharedLockers: shareds.NewLocalLockers(),
		sharedStore:   sharedStore,
		handleTimeout: time.Duration(handleTimeoutSeconds) * time.Second,
		discovery:     nil,
		deployed:      make(map[string]*endpoint),
	}

	v.rt.discovery = v
	return
}

type Endpoints struct {
	rt            *runtimes
	log           logs.Logger
	appId         string
	appVersion    versions.Version
	running       *flags.Flag
	worker        Workers
	handleTimeout time.Duration
	barrier       Barrier
	sharedLockers shared.Lockers
	sharedStore   shared.Store
	discovery     EndpointDiscovery
	deployed      map[string]*endpoint
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
		id:           "",
		native:       false,
		versionRange: make([]versions.Version, 0, 1),
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	versionMatched := true
	if len(opt.versionRange) > 0 {
		versionMatched = e.appVersion.Between(opt.versionRange[0], opt.versionRange[1])
	}
	if opt.id != "" {
		if opt.id == e.appId && versionMatched {
			endpoint, has = e.deployed[service]
			return
		}
		if e.discovery != nil && CanAccessInternal(ctx) && !opt.native {
			endpoint, has = e.discovery.Get(ctx, service, options...)
		}
	} else {
		endpoint, has = e.deployed[service]
		if has && versionMatched {
			return
		}
		if e.discovery != nil && CanAccessInternal(ctx) && !opt.native {
			endpoint, has = e.discovery.Get(ctx, service, options...)
		}
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
	// todo listen http
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
	// todo close http
	// todo close http handler
	for _, ep := range e.deployed {
		ep.svc.Close()
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

type Endpoint interface {
	Name() (name string)
	Internal() (ok bool)
	Document() (document Document)
	Request(ctx context.Context, request Request) (result Result)
}

type endpoint struct {
	appId         string
	running       *flags.Flag
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

func (e *endpoint) Internal() (ok bool) {
	ok = e.svc.Internal()
	return
}

func (e *endpoint) Document() (document Document) {
	document = e.svc.Document()
	return
}

func (e *endpoint) Request(ctx context.Context, req Request) (result Result) {
	ctx = withRuntime(ctx, e.appId, e.log, e.worker, e.discovery, e.barrier, e.sharedLockers, e.sharedStore, e.running)
	ctx = withRequest(ctx, req)
	ctx = withTracer(ctx)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, e.handleTimeout)
	// todo 双重barrier，1. request.code+deviceId，2，request.code

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

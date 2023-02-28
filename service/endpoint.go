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
	"github.com/aacfactory/fns/service/internal/logger"
	"github.com/aacfactory/fns/service/internal/procs"
	"github.com/aacfactory/fns/service/shared"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type EndpointsOptions struct {
	AppId           string
	AppName         string
	AppVersion      versions.Version
	AutoMaxProcsMin int
	AutoMaxProcsMax int
	Http            Http
	Config          *Config
}

func NewEndpoints(options EndpointsOptions) (v *Endpoints, err error) {
	// log >>>
	logOptions := logger.LogOptions{
		Name: options.AppName,
	}
	if options.Config.Log != nil {
		logOptions.Color = options.Config.Log.Color
		logOptions.Formatter = options.Config.Log.Formatter
		logOptions.Level = options.Config.Log.Level
	}
	log, logErr := logger.NewLog(logOptions)
	if logErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(logErr)
		return
	}
	// log <<<
	// procs
	goprocs := procs.New(procs.Options{
		Log: log,
		Min: options.AutoMaxProcsMin,
		Max: options.AutoMaxProcsMax,
	})
	// workers >>>
	rtConfig := options.Config.Runtime
	if rtConfig == nil {
		rtConfig = &RuntimeConfig{}
	}
	maxWorkers := rtConfig.MaxWorkers
	if maxWorkers < 1 {
		maxWorkers = 256 * 1024
	}
	maxIdleWorkerSeconds := rtConfig.WorkerMaxIdleSeconds
	if maxIdleWorkerSeconds < 1 {
		maxIdleWorkerSeconds = 60
	}
	worker := workers.New(workers.MaxWorkers(maxWorkers), workers.MaxIdleWorkerDuration(time.Duration(maxIdleWorkerSeconds)*time.Second))
	// workers <<<

	// http >>>
	if options.Http == nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("http is required"))
		return
	}
	// http <<<

	// cluster
	var cluster Cluster
	var sharedStore shared.Store
	var sharedLockers shared.Lockers
	var barrier Barrier
	var registrations *Registrations
	if options.Config.Cluster != nil {
		// cluster >>>
		kind := strings.TrimSpace(options.Config.Cluster.Kind)
		builder, hasBuilder := getClusterBuilder(kind)
		if !hasBuilder {
			err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("kind of cluster is not found").WithMeta("kind", kind))
			return
		}
		cluster, err = builder(ClusterBuilderOptions{
			Config:     options.Config.Cluster,
			Log:        log.With("cluster", kind),
			AppId:      options.AppId,
			AppName:    options.AppName,
			AppVersion: options.AppVersion,
		})
		if err != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(err).WithMeta("kind", kind)
			return
		}
		// cluster <<<
		sharedStore = cluster.Shared().Store()
		sharedLockers = cluster.Shared().Lockers()
		barrier = &sharedBarrier{
			store: sharedStore,
		}
		// registrations
		registrations = &Registrations{
			id:     options.AppId,
			locker: sync.Mutex{},
			values: sync.Map{},
		}
	} else {
		// shared store >>>
		sharedMemSizeStr := strings.TrimSpace(rtConfig.LocalSharedStoreCacheSize)
		if sharedMemSizeStr == "" {
			sharedMemSizeStr = "64M"
		}
		sharedMemSize, sharedMemSizeErr := bytex.ToBytes(sharedMemSizeStr)
		if sharedMemSizeErr != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(sharedMemSizeErr)
			return
		}
		sharedStore, err = shared.NewLocalStore(int64(sharedMemSize))
		if err != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(err)
			return
		}
		// shared store <<<
		// shared lockers >>>
		sharedLockers = shared.NewLocalLockers()
		// shared lockers <<<
		// barrier >>>
		barrier = defaultBarrier()
		// barrier <<<
	}
	// timeout
	handleTimeoutSeconds := options.Config.Runtime.HandleTimeoutSeconds
	if handleTimeoutSeconds < 1 {
		handleTimeoutSeconds = 10
	}
	v = &Endpoints{
		rt: &runtimes{
			appId:         options.AppId,
			running:       flags.New(false),
			log:           log,
			worker:        worker,
			discovery:     nil,
			barrier:       barrier,
			sharedLockers: sharedLockers,
			sharedStore:   sharedStore,
		},
		autoMaxProcs:  goprocs,
		log:           log,
		handleTimeout: time.Duration(handleTimeoutSeconds) * time.Second,
		deployed:      make(map[string]*endpoint),
		registrations: registrations,
		http:          options.Http,
		httpConfig:    options.Config.Http,
		cluster:       cluster,
	}

	v.rt.discovery = v
	return
}

type Endpoints struct {
	log           logs.Logger
	rt            *runtimes
	autoMaxProcs  *procs.AutoMaxProcs
	handleTimeout time.Duration
	deployed      map[string]*endpoint
	registrations *Registrations
	http          Http
	httpConfig    *HttpConfig
	cluster       Cluster
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
		versionMatched = e.rt.appVersion.Between(opt.versionRange[0], opt.versionRange[1])
	}
	if opt.id != "" {
		if opt.id == e.rt.appId && versionMatched {
			endpoint, has = e.deployed[service]
			return
		}
		if e.registrations != nil && CanAccessInternal(ctx) && !opt.native {
			registration, fetched := e.registrations.Get(opt.id, service)
			if !fetched {
				return
			}
			versionMatched = registration.version.Between(opt.versionRange[0], opt.versionRange[1])
			if !versionMatched {
				return
			}
			endpoint = registration
			has = true
			return
		}
	} else {
		endpoint, has = e.deployed[service]
		if has && versionMatched {
			return
		}
		if e.registrations != nil && CanAccessInternal(ctx) && !opt.native {
			registration, fetched := e.registrations.Get("", service)
			if !fetched {
				return
			}
			versionMatched = registration.version.Between(opt.versionRange[0], opt.versionRange[1])
			if !versionMatched {
				return
			}
			endpoint = registration
			has = true
			return
		}
	}
	return
}

func (e *Endpoints) Mount(svc Service) {
	e.deployed[svc.Name()] = &endpoint{
		rt:            e.rt,
		handleTimeout: e.handleTimeout,
		svc:           svc,
	}
}

func (e *Endpoints) Listen() (err error) {
	e.autoMaxProcs.Enable()
	// todo cluster join

	// todo listen registrations

	// todo create http and listen

	// handlers

	// http

	// listen endpoint after cluster cause the endpoint may use cluster
	errCh := make(chan error, 8)
	lns := 0
	closed := int64(0)
	for _, ep := range e.deployed {
		ln, ok := ep.svc.(Listenable)
		if !ok {
			continue
		}
		lns++
		ctx := withRuntime(context.TODO(), e.rt)
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
	if lns > 0 {
		select {
		case lnErr := <-errCh:
			atomic.AddInt64(&closed, 1)
			err = errors.Warning("fns: endpoints listen failed").WithCause(lnErr)
			break
		case <-time.After(time.Duration(lns*3) * time.Second):
			break
		}
		close(errCh)
	}
	e.rt.running.On()
	return
}

func (e *Endpoints) Running() (ok bool) {
	ok = e.rt.running.IsOn()
	return
}

func (e *Endpoints) Close() {
	e.rt.running.Off()
	// todo close http
	// todo close http handler
	for _, ep := range e.deployed {
		ep.svc.Close()
	}
	e.autoMaxProcs.Reset()
}

// +-------------------------------------------------------------------------------------------------------------------+

type Endpoint interface {
	Name() (name string)
	Internal() (ok bool)
	Document() (document Document)
	Request(ctx context.Context, r Request) (result Result)
	RequestSync(ctx context.Context, r Request) (result interface{}, has bool, err errors.CodeError)
}

type endpoint struct {
	handleTimeout time.Duration
	svc           Service
	rt            *runtimes
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

func (e *endpoint) Request(ctx context.Context, r Request) (result Result) {
	ctx = withRuntime(ctx, e.rt)
	ctx = withRequest(ctx, r)
	ctx = withTracer(ctx)
	fr := NewResult()
	if !e.rt.worker.Dispatch(ctx, newFnTask(e.svc, e.rt.barrier, r, fr, e.handleTimeout)) {
		serviceName, fnName := r.Fn()
		if ctx.Err() != nil {
			fr.Failed(errors.Timeout("fns: workers handle timeout").WithMeta("fns", "timeout").WithMeta("service", serviceName).WithMeta("fn", fnName))
		} else {
			fr.Failed(errors.NotAcceptable("fns: service is overload").WithMeta("fns", "overload").WithMeta("service", serviceName).WithMeta("fn", fnName))
		}
	}
	result = fr
	tryReportTracer(ctx)
	return
}

func (e *endpoint) RequestSync(ctx context.Context, r Request) (result interface{}, has bool, err errors.CodeError) {
	fr := e.Request(ctx, r)
	result, has, err = fr.Value(ctx)
	return
}

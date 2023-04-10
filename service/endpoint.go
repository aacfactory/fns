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
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/documents"
	"github.com/aacfactory/fns/service/internal/commons/flags"
	"github.com/aacfactory/fns/service/internal/logger"
	"github.com/aacfactory/fns/service/internal/procs"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type TransportOptions struct {
	Middlewares []TransportMiddleware
	Handlers    []TransportHandler
}

type EndpointsOptions struct {
	AppId      string
	AppName    string
	AppVersion versions.Version
	Transport  *TransportOptions
	Proxy      *TransportOptions
	Config     configures.Config
}

func NewEndpoints(options EndpointsOptions) (v *Endpoints, err error) {
	embeds := make([]Service, 0, 1)
	// config
	config := &Config{}
	configErr := options.Config.As(config)
	if configErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(configErr)
		return
	}
	if options.Transport == nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("transport is required"))
		return
	}

	// log >>>
	logConfig := config.Log
	logOptions := logger.LogOptions{
		Name: options.AppName,
	}
	if logConfig != nil {
		logOptions.Color = config.Log.Color
		logOptions.Formatter = config.Log.Formatter
		logOptions.Level = config.Log.Level
	}
	log, logErr := logger.NewLog(logOptions)
	if logErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(logErr)
		return
	}
	log = log.With("app", options.AppName).With("appId", options.AppId)
	// log <<<

	runtimeConfig := config.Runtime
	if runtimeConfig == nil {
		runtimeConfig = &RuntimeConfig{}
	}
	// secret key
	secretKey := strings.TrimSpace(runtimeConfig.SecretKey)
	if secretKey == "" {
		secretKey = secret.DefaultSignerKey
	}

	// procs
	goprocs := procs.New(procs.Options{
		Log: log,
		Min: runtimeConfig.AutoMaxProcs.Min,
		Max: runtimeConfig.AutoMaxProcs.Max,
	})
	// workers >>>

	maxWorkers := runtimeConfig.MaxWorkers
	if maxWorkers < 1 {
		maxWorkers = 256 * 1024
	}
	maxIdleWorkerSeconds := runtimeConfig.WorkerMaxIdleSeconds
	if maxIdleWorkerSeconds < 1 {
		maxIdleWorkerSeconds = 60
	}
	worker := workers.New(workers.MaxWorkers(maxWorkers), workers.MaxIdleWorkerDuration(time.Duration(maxIdleWorkerSeconds)*time.Second))
	// workers <<<

	// cluster
	var cluster Cluster
	var shared Shared
	var barrier Barrier
	clusterFetchMembersInterval := time.Duration(0)
	if config.Cluster != nil {
		// cluster >>>
		if config.Cluster.FetchMembersInterval != "" {
			clusterFetchMembersInterval, err = time.ParseDuration(strings.TrimSpace(config.Cluster.FetchMembersInterval))
			if err != nil {
				err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("fetchMembersInterval must be time.Duration format")).WithCause(err)
				return
			}
		}
		if clusterFetchMembersInterval < 1*time.Second {
			clusterFetchMembersInterval = 10 * time.Second
		}

		kind := strings.TrimSpace(config.Cluster.Kind)
		if kind == devClusterBuilderName && options.Proxy != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("cannot use dev cluster in proxy transport")).WithCause(err)
			return
		}
		builder, hasBuilder := getClusterBuilder(kind)
		if !hasBuilder {
			err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("kind of cluster is not found").WithMeta("kind", kind))
			return
		}
		if config.Cluster.Options == nil || len(config.Cluster.Options) == 0 {
			config.Cluster.Options = []byte{'{', '}'}
		}
		clusterOptionConfig, clusterOptionConfigErr := configures.NewJsonConfig(config.Cluster.Options)
		if clusterOptionConfigErr != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("cluster: build cluster options config failed")).WithCause(clusterOptionConfigErr).WithMeta("kind", kind)
			return
		}
		cluster, err = builder(ClusterBuilderOptions{
			Config:     clusterOptionConfig,
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
		shared = cluster.Shared()
		if config.Cluster.Shared != nil {
			if config.Cluster.Shared.BarrierDisabled {
				barrier = defaultBarrier()
			} else {
				barrierTTL := 100 * time.Millisecond
				if config.Cluster.Shared.BarrierTTLMilliseconds > 0 {
					barrierTTL = time.Duration(config.Cluster.Shared.BarrierTTLMilliseconds) * time.Millisecond
				}
				barrier = clusterBarrier(shared, barrierTTL)
			}
		}
	} else {
		// shared >>>
		shared, err = newLocalShared()
		if err != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(err)
			return
		}
		// shared <<<
		// barrier >>>
		barrier = defaultBarrier()
		// barrier <<<
	}
	// timeout
	handleTimeoutSeconds := runtimeConfig.HandleTimeoutSeconds
	if handleTimeoutSeconds < 1 {
		handleTimeoutSeconds = 10
	}
	// endpoints
	v = &Endpoints{
		log: log,
		rt: &Runtime{
			appId:      options.AppId,
			appName:    options.AppName,
			appVersion: options.AppVersion,
			status: &Status{
				flag: flags.New(false),
			},
			log:       log,
			worker:    worker,
			discovery: nil,
			barrier:   barrier,
			shared:    shared,
			signer:    secret.NewSigner(bytex.FromString(secretKey)),
		},
		autoMaxProcs:  goprocs,
		handleTimeout: time.Duration(handleTimeoutSeconds) * time.Second,
		config:        options.Config,
		deployed:      make(map[string]*endpoint),
		deployedCHS:   newDeployed(),
		registrations: nil,
		transport:     nil,
		proxy:         nil,
		closers:       make([]io.Closer, 0, 1),
		cluster:       cluster,
		closeCh:       make(chan struct{}, 1),
	}
	v.rt.discovery = v

	// transports >>>
	transport, transportClosers, transportErr := createService(config.Transport, v.deployedCHS.acquire(), v.rt, options.Transport.Middlewares, options.Transport.Handlers)
	if transportErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(transportErr)
		return
	}
	v.closers = append(v.closers, transportClosers...)
	v.transport = transport
	for _, middleware := range options.Transport.Middlewares {
		v.closers = append(v.closers, middleware)
		servicesSupplier, ok := middleware.(ServicesSupplier)
		if ok && servicesSupplier.Services() != nil {
			embeds = append(embeds, servicesSupplier.Services()...)
		}
	}
	for _, handlers := range options.Transport.Handlers {
		servicesSupplier, ok := handlers.(ServicesSupplier)
		if ok && servicesSupplier.Services() != nil {
			embeds = append(embeds, servicesSupplier.Services()...)
		}
	}
	if cluster != nil {
		requestCacheDefaultTTL, hasRequestCacheDefaultTTL := config.Transport.GetRequestCache()
		if !hasRequestCacheDefaultTTL {
			requestCacheDefaultTTL = -1
		}
		if requestCacheDefaultTTL < 1 {
			requestCacheDefaultTTL = 30 * time.Minute
		}
		v.registrations = newRegistrations(
			v.rt.log.With("discovery", "registrations"),
			v.rt.appId, v.rt.appName, v.rt.appVersion,
			cluster, v.rt.worker, v.transport, v.rt.signer, v.handleTimeout,
			clusterFetchMembersInterval, requestCacheDefaultTTL,
		)
	}
	// proxy
	if options.Proxy != nil {
		proxy, proxyClosers, proxyErr := createProxy(config.Proxy, v.deployedCHS.acquire(), v.rt, v.registrations, options.Proxy.Middlewares, options.Proxy.Handlers)
		if proxyErr != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(proxyErr)
			return
		}
		v.closers = append(v.closers, proxyClosers...)
		v.proxy = proxy
		for _, middleware := range options.Proxy.Middlewares {
			servicesSupplier, ok := middleware.(ServicesSupplier)
			if ok && servicesSupplier.Services() != nil {
				embeds = append(embeds, servicesSupplier.Services()...)
			}
		}
		for _, handlers := range options.Proxy.Handlers {
			servicesSupplier, ok := handlers.(ServicesSupplier)
			if ok && servicesSupplier.Services() != nil {
				embeds = append(embeds, servicesSupplier.Services()...)
			}
		}
	}
	// transports <<<

	// embeds
	if len(embeds) > 0 {
		for _, embed := range embeds {
			deployErr := v.Deploy(embed)
			if deployErr != nil {
				err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("deploy embed service failed").WithCause(deployErr))
				return
			}
		}
	}
	return
}

type Endpoints struct {
	log           logs.Logger
	rt            *Runtime
	autoMaxProcs  *procs.AutoMaxProcs
	handleTimeout time.Duration
	config        configures.Config
	deployed      map[string]*endpoint
	deployedCHS   *deployed
	registrations *Registrations
	transport     transports.Transport
	proxy         transports.Transport
	closers       []io.Closer
	cluster       Cluster
	closeCh       chan struct{}
}

func (e *Endpoints) Log() (log logs.Logger) {
	log = e.rt.log
	return
}

func (e *Endpoints) Runtime() (rt *Runtime) {
	rt = e.rt
	return
}

func (e *Endpoints) Get(ctx context.Context, service string, options ...EndpointDiscoveryGetOption) (v Endpoint, has bool) {
	if service == "" {
		return
	}
	opt := &EndpointDiscoveryGetOptions{
		id:              "",
		native:          false,
		requestVersions: nil,
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	rvs := opt.requestVersions
	if rvs == nil {
		req, hasRequest := GetRequest(ctx)
		if hasRequest {
			rvs = req.AcceptedVersions()
		}
	}
	if opt.id != "" {
		if opt.id == e.rt.appId {
			v, has = e.deployed[service]
			return
		}
		if e.registrations != nil && CanAccessInternal(ctx) && !opt.native {
			v, has = e.registrations.GetExact(service, opt.id)
			return
		}
	} else {
		v, has = e.deployed[service]
		if has {
			if rvs == nil || rvs.Accept(service, e.rt.appVersion) {
				return
			}
		}
		has = false
		if opt.native {
			return
		}
		if e.registrations != nil && CanAccessInternal(ctx) {
			v, has = e.registrations.Get(service, rvs)
			return
		}
	}
	return
}

func (e *Endpoints) Deploy(svc Service) (err error) {
	name := strings.TrimSpace(svc.Name())
	serviceConfig, hasConfig := e.config.Node(name)
	if !hasConfig {
		serviceConfig, _ = configures.NewJsonConfig([]byte("{}"))
	}
	buildErr := svc.Build(Options{
		AppId:      e.rt.appId,
		AppName:    e.rt.appName,
		AppVersion: e.rt.appVersion,
		Log:        e.log.With("fns", "service").With("service", name),
		Config:     serviceConfig,
	})
	if buildErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: endpoints deploy %s service failed", name)).WithMeta("service", name).WithCause(buildErr)
		return
	}
	ep := &endpoint{
		rt:            e.rt,
		handleTimeout: e.handleTimeout,
		svc:           svc,
		pool:          sync.Pool{},
	}
	ep.pool.New = func() any {
		return newFnTask(svc, e.rt.barrier, e.handleTimeout, func(task *fnTask) {
			ep.release(task)
		})
	}
	e.deployed[svc.Name()] = ep

	return
}

func (e *Endpoints) Listen(ctx context.Context) (err error) {
	e.rt.status.flag.HalfOn()
	e.autoMaxProcs.Enable()
	e.deployedCHS.publish(e.deployed)
	// cluster join
	if e.cluster != nil {
		joinErr := e.cluster.Join(ctx)
		if joinErr != nil {
			e.Close(ctx)
			err = errors.Warning("fns: endpoints listen failed").WithCause(joinErr)
			return
		}
		e.fetchRegistrations()
	}
	// transport listen
	transportListenErrCh := make(chan error, 2)
	go func(srv transports.Transport, ch chan error) {
		listenErr := srv.ListenAndServe()
		if listenErr != nil {
			ch <- errors.Warning("fns: endpoints listen failed").WithCause(listenErr)
			close(ch)
		}
	}(e.transport, transportListenErrCh)
	if e.proxy != nil {
		go func(srv transports.Transport, ch chan error) {
			listenErr := srv.ListenAndServe()
			if listenErr != nil {
				ch <- errors.Warning("fns: endpoints listen failed").WithCause(listenErr)
				close(ch)
			}
		}(e.proxy, transportListenErrCh)
	}
	select {
	case <-time.After(3 * time.Second):
		break
	case transportErr := <-transportListenErrCh:
		e.Close(ctx)
		err = errors.Warning("fns: endpoints listen failed").WithCause(transportErr)
		return
	}

	// listen endpoint after cluster cause the endpoint may use cluster
	serviceListenErrCh := make(chan error, 8)
	lns := 0
	closed := int64(0)
	for _, ep := range e.deployed {
		ln, ok := ep.svc.(Listenable)
		if !ok {
			continue
		}
		lns++
		go func(ctx context.Context, ln Listenable, errCh chan error) {
			if atomic.LoadInt64(&closed) == 1 {
				return
			}
			lnErr := ln.Listen(ctx)
			if lnErr != nil {
				lnErr = errors.Warning(fmt.Sprintf("fns: %s listen falied", ln.Name())).WithCause(lnErr).WithMeta("service", ln.Name())
				if atomic.LoadInt64(&closed) == 0 {
					errCh <- lnErr
				}
			}
		}(e.rt.SetIntoContext(context.TODO()), ln, serviceListenErrCh)
	}
	if lns > 0 {
		select {
		case serviceListenErr := <-serviceListenErrCh:
			atomic.AddInt64(&closed, 1)
			e.Close(ctx)
			err = errors.Warning("fns: endpoints listen failed").WithCause(serviceListenErr)
			return
		case <-time.After(time.Duration(lns) * time.Second):
			break
		}
	}
	e.rt.status.flag.On()
	return
}

func (e *Endpoints) Running() (ok bool) {
	ok = !e.rt.status.Closed()
	return
}

func (e *Endpoints) Close(ctx context.Context) {
	e.closeCh <- struct{}{}
	close(e.closeCh)

	e.rt.status.flag.Off()

	for _, closer := range e.closers {
		_ = closer.Close()
	}
	if e.cluster != nil {
		_ = e.cluster.Leave(ctx)
	}
	if e.registrations != nil {
		e.registrations.Close()
	}

	_ = e.transport.Close()
	if e.proxy != nil {
		_ = e.proxy.Close()
	}

	for _, ep := range e.deployed {
		ep.svc.Close()
	}
	e.autoMaxProcs.Reset()
}

func (e *Endpoints) fetchRegistrations() {
	ctx, cancel := context.WithCancel(context.TODO())
	go e.registrations.Refresh(ctx)
	go func(closeCh chan struct{}, cancel context.CancelFunc) {
		<-closeCh
		cancel()
	}(e.closeCh, cancel)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Endpoint interface {
	Name() (name string)
	Internal() (ok bool)
	Document() (document *documents.Document)
	Request(ctx context.Context, r Request) (future Future)
	RequestSync(ctx context.Context, r Request) (result FutureResult, err errors.CodeError)
}

type endpoint struct {
	handleTimeout time.Duration
	svc           Service
	rt            *Runtime
	pool          sync.Pool
}

func (e *endpoint) Name() (name string) {
	name = e.svc.Name()
	return
}

func (e *endpoint) Internal() (ok bool) {
	ok = e.svc.Internal()
	return
}

func (e *endpoint) Document() (document *documents.Document) {
	document = e.svc.Document()
	return
}

func (e *endpoint) Request(ctx context.Context, r Request) (future Future) {
	ctx = withTracer(ctx, r.Id())
	ctx = withRequest(ctx, r)
	promise, fr := NewFuture()
	task := e.acquire()
	task.begin(r, promise)
	if !e.rt.worker.Dispatch(ctx, task) {
		serviceName, fnName := r.Fn()
		if ctx.Err() != nil {
			promise.Failed(errors.Timeout("fns: workers handle timeout").WithMeta("fns", "timeout").WithMeta("service", serviceName).WithMeta("fn", fnName))
		} else {
			promise.Failed(ErrServiceOverload.WithMeta("service", serviceName).WithMeta("fn", fnName))
		}
		e.release(task)
	}
	future = fr
	tryReportTracer(ctx)
	return
}

func (e *endpoint) RequestSync(ctx context.Context, r Request) (result FutureResult, err errors.CodeError) {
	future := e.Request(ctx, r)
	result, err = future.Get(ctx)
	return
}

func (e *endpoint) acquire() (task *fnTask) {
	v := e.pool.Get()
	if v != nil {
		task = v.(*fnTask)
		return
	}
	task = newFnTask(e.svc, e.rt.barrier, e.handleTimeout, func(task *fnTask) {
		e.release(task)
	})
	return
}

func (e *endpoint) release(task *fnTask) {
	task.end()
	e.pool.Put(task)
	return
}

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
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/flags"
	"github.com/aacfactory/fns/commons/procs"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/logger"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type TransportOptions struct {
	Transport   transports.Transport
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
		secretKey = "+fns-"
	}

	// procs
	goprocs := procs.New(runtimeConfig.AutoMaxProcs.Min, runtimeConfig.AutoMaxProcs.Max)
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
	var shared shareds.Shared
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
	} else {
		// shared >>>
		shared, err = shareds.Local()
		if err != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(err)
			return
		}
		// shared <<<
	}
	// barrier >>>
	barrier = defaultBarrier()
	// barrier <<<
	// timeout
	handleTimeoutSeconds := runtimeConfig.HandleTimeoutSeconds
	if handleTimeoutSeconds < 1 {
		handleTimeoutSeconds = 10
	}
	// endpoints
	v = &Endpoints{
		log: log,
		rt: &Runtime{
			appId:       options.AppId,
			appName:     options.AppName,
			appVersion:  options.AppVersion,
			appServices: make([]NamePlate, 0, 1),
			status: &Status{
				flag: flags.New(false),
			},
			log:       log,
			worker:    worker,
			discovery: nil,
			barrier:   barrier,
			shared:    shared,
			signer:    signatures.HMAC(bytex.FromString(secretKey)),
		},
		autoMaxProcs:     goprocs,
		handleTimeout:    time.Duration(handleTimeoutSeconds) * time.Second,
		config:           options.Config,
		deployed:         make(map[string]*endpoint),
		registrations:    nil,
		serviceTransport: nil,
		proxyTransport:   nil,
		cluster:          cluster,
		closeCh:          make(chan struct{}, 1),
	}
	v.rt.discovery = v

	// transports >>>
	serviceTransport, serviceTransportErr := createServiceTransport(config.Transport, v.rt, options.Transport.Middlewares, options.Transport.Handlers)
	if serviceTransportErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(serviceTransportErr)
		return
	}
	v.serviceTransport = serviceTransport
	v.rt.appPort = serviceTransport.Port()
	embeds = append(embeds, serviceTransport.services()...)
	if cluster != nil {
		v.registrations = newRegistrations(
			v.rt.log.With("discovery", "registrations"),
			v.rt.appId, v.rt.appName, v.rt.appVersion,
			cluster, v.rt.worker, serviceTransport.transport, v.rt.signer, v.handleTimeout,
			clusterFetchMembersInterval,
		)
	}
	// proxy
	if options.Proxy != nil {
		proxyTransport, proxyTransportErr := createProxyTransport(config.Proxy, v.rt, v.registrations, options.Proxy.Middlewares, options.Proxy.Handlers)
		if proxyTransportErr != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(proxyTransportErr)
			return
		}
		v.proxyTransport = serviceTransport
		embeds = append(embeds, proxyTransport.services()...)
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

type Endpoints1 struct {
	log              logs.Logger
	rt               *Runtime
	autoMaxProcs     *procs.AutoMaxProcs
	handleTimeout    time.Duration
	config           configures.Config
	deployed         map[string]*endpoint
	registrations    *Registrations
	serviceTransport *Transport
	proxyTransport   *Transport
	cluster          Cluster
	closeCh          chan struct{}
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

func (e *Endpoints) Listen(ctx context.Context) (err error) {
	e.rt.status.flag.HalfOn()
	e.autoMaxProcs.Enable()
	// cluster fetch
	if e.cluster != nil {
		e.fetchRegistrations()
	}
	// transport listen
	err = e.serviceTransport.Listen(ctx)
	if err != nil {
		err = errors.Warning("fns: endpoints listen failed").WithCause(err)
		return
	}
	if e.proxyTransport != nil {
		err = e.proxyTransport.Listen(ctx)
		if err != nil {
			err = errors.Warning("fns: endpoints listen failed").WithCause(err)
			return
		}
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
		}(e.rt.SetIntoContext(context.TODO()), ln, serviceListenErrCh) // runtime 在外面的ctx中，
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
	// cluster join
	if e.cluster != nil {
		joinErr := e.cluster.Join(ctx)
		if joinErr != nil {
			e.Close(ctx)
			err = errors.Warning("fns: endpoints listen failed").WithCause(joinErr)
			return
		}
		e.Close(ctx)
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
	// leave
	if e.cluster != nil {
		_ = e.cluster.Leave(ctx)
	}
	// close transport
	_ = e.serviceTransport.Close()
	if e.proxyTransport != nil {
		_ = e.proxyTransport.Close()
	}
	// close registrations
	if e.registrations != nil {
		e.registrations.Close()
	}
	// close services
	for _, ep := range e.deployed {
		ep.svc.Close()
	}
	// reset max proc
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
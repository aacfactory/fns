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
	"crypto/tls"
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
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type EndpointsOptions struct {
	OpenApiVersion   string
	AppId            string
	AppName          string
	AppVersion       versions.Version
	ProxyMode        bool
	Http             Http
	HttpHandlers     []HttpHandler
	HttpInterceptors []HttpInterceptor
	Config           configures.Config
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
	if options.Http == nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("http is required"))
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
	clusterDevMode := false
	clusterProxyAddress := ""
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
		if config.Cluster.DevMode != nil {
			clusterDevMode = true
			clusterProxyAddress = config.Cluster.DevMode.ProxyAddress
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
		sharedMemSizeStr := strings.TrimSpace(runtimeConfig.LocalSharedStoreCacheSize)
		if sharedMemSizeStr == "" {
			sharedMemSizeStr = "64M"
		}
		sharedMemSize, sharedMemSizeErr := bytex.ToBytes(sharedMemSizeStr)
		if sharedMemSizeErr != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(sharedMemSizeErr)
			return
		}
		shared, err = newLocalShared(int64(sharedMemSize))
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
	handleTimeoutSeconds := config.Runtime.HandleTimeoutSeconds
	if handleTimeoutSeconds < 1 {
		handleTimeoutSeconds = 10
	}
	// endpoints
	v = &Endpoints{
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
		config:                   options.Config,
		autoMaxProcs:             goprocs,
		log:                      log,
		handleTimeout:            time.Duration(handleTimeoutSeconds) * time.Second,
		deployed:                 make(map[string]*endpoint),
		deployedCHS:              newDeployed(),
		registrations:            nil,
		http:                     options.Http,
		httpHandlers:             nil,
		cluster:                  cluster,
		clusterNodeFetchInterval: clusterFetchMembersInterval,
		clusterProxyAddress:      clusterProxyAddress,
		closeCh:                  make(chan struct{}, 1),
	}
	v.rt.discovery = v
	if cluster != nil {
		v.registrations = &Registrations{
			id:      v.rt.appId,
			values:  sync.Map{},
			dialer:  v.http,
			signer:  v.rt.signer,
			worker:  v.rt.worker,
			timeout: v.handleTimeout,
		}
	}
	// http >>>
	httpConfig := config.Http
	if httpConfig == nil {
		httpConfig = &HttpConfig{
			Port:     80,
			Cors:     nil,
			TLS:      nil,
			Options:  nil,
			Handlers: nil,
		}
	}
	handlersConfigBytes := httpConfig.Handlers
	if handlersConfigBytes == nil || len(handlersConfigBytes) == 0 {
		handlersConfigBytes = []byte{'{', '}'}
	}
	handlersConfig, handlersConfigErr := configures.NewJsonConfig(handlersConfigBytes)
	if handlersConfigErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(handlersConfigErr)
		return
	}
	handlers, handlersErr := NewHttpHandlers(HandlersOptions{
		AppId:      v.rt.appId,
		AppName:    v.rt.appName,
		AppVersion: v.rt.appVersion,
		Log:        v.rt.log.With("http", "handlers"),
		Config:     handlersConfig,
		Discovery:  v,
		Status:     v.rt.status,
	})
	if handlersErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(handlersErr)
		return
	}
	v.httpHandlers = handlers

	appendHandlerErr := handlers.Append(newServicesHandler(servicesHandlerOptions{
		Signer:     v.rt.signer,
		DeployedCh: v.deployedCHS.acquire(),
	}))
	if appendHandlerErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(appendHandlerErr)
		return
	}
	if options.HttpHandlers != nil && len(options.HttpHandlers) > 0 {
		for _, handler := range options.HttpHandlers {
			if handler == nil {
				continue
			}
			appendHandlerErr = handlers.Append(handler)
			if appendHandlerErr != nil {
				err = errors.Warning("fns: create endpoints failed").WithCause(appendHandlerErr)
				return
			}
			handlerWithService, ok := handler.(HttpHandlerWithServices)
			if ok {
				servicesOfHandler := handlerWithService.Services()
				if servicesOfHandler != nil && len(servicesOfHandler) > 0 {
					embeds = append(embeds, servicesOfHandler...)
				}
			}
		}
	}
	if options.HttpInterceptors != nil && len(options.HttpInterceptors) > 0 {
		for _, interceptor := range options.HttpInterceptors {
			if interceptor == nil {
				continue
			}
			appendHandlerErr = handlers.AppendInterceptor(interceptor)
			if appendHandlerErr != nil {
				err = errors.Warning("fns: create endpoints failed").WithCause(appendHandlerErr)
				return
			}
			interceptorWithService, ok := interceptor.(HttpInterceptorWithServices)
			if ok {
				servicesOfInterceptorWithService := interceptorWithService.Services()
				if servicesOfInterceptorWithService != nil && len(servicesOfInterceptorWithService) > 0 {
					embeds = append(embeds, servicesOfInterceptorWithService...)
				}
			}
		}
	}
	httpHandler := handlers.Build()
	if httpConfig.Cors != nil {
		httpHandler = newCorsHandler(httpConfig.Cors).Handler(httpHandler)
	}
	var serverTLS *tls.Config
	var clientTLS *tls.Config
	if httpConfig.TLS != nil {
		serverTLS, clientTLS, err = httpConfig.TLS.Config()
		if err != nil {
			err = errors.Warning("fns: create endpoints failed").WithCause(err)
			return
		}
	}
	if httpConfig.Options == nil || len(httpConfig.Options) < 2 {
		httpConfig.Options = []byte{'{', '}'}
	}
	httpConfigOptions, httpConfigOptionsErr := configures.NewJsonConfig(httpConfig.Options)
	if httpConfigOptionsErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(httpConfigOptionsErr)
		return
	}
	buildErr := v.http.Build(HttpOptions{
		Port:      httpConfig.Port,
		ServerTLS: serverTLS,
		ClientTLS: clientTLS,
		Handler:   httpHandler,
		Log:       v.rt.log.With("http", v.http.Name()),
		Options:   httpConfigOptions,
	})
	if buildErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(buildErr)
		return
	}
	// http <<<
	// dev >>
	if clusterDevMode {
		// todo 在构建cluster的时候代理，取出cluster和http以及clusterProxyAddress参数。
		// 因为自身http的dialer不是proxy的，所以不再proxy上开dev，而是在http上加dev handler。所以走的是非proxy的port。
		// 实现为在services的internal handler里判断是否是dev。如果是，则discovery不带native，反之带。
		v.cluster = newDevProxyCluster(v.rt.appId, v.cluster, v.clusterProxyAddress, v.http, bytex.FromString(secretKey))
		// todo services 里的
	}
	// dev <<<
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
	log                      logs.Logger
	rt                       *Runtime
	autoMaxProcs             *procs.AutoMaxProcs
	handleTimeout            time.Duration
	config                   configures.Config
	deployed                 map[string]*endpoint
	deployedCHS              *deployed
	registrations            *Registrations
	transport                Transport
	transportHandlers        *TransportHandlers
	cluster                  Cluster
	clusterNodeFetchInterval time.Duration
	clusterProxyAddress      string
	closeCh                  chan struct{}
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
			e.rt.status.flag.Off()
			err = errors.Warning("fns: endpoints listen failed").WithCause(joinErr)
			return
		}
		nodes, nodesErr := listMembers(ctx, e.cluster, e.rt.appId, e.rt.appName, e.rt.appVersion)
		if nodesErr != nil {
			e.rt.status.flag.Off()
			err = errors.Warning("fns: endpoints listen failed").WithCause(errors.Warning("fns: endpoints get nodes from cluster failed")).WithCause(nodesErr)
			return
		}
		// registrations
		mergeErr := e.registrations.MergeNodes(nodes)
		if mergeErr != nil {
			e.rt.status.flag.Off()
			err = errors.Warning("fns: endpoints listen failed").WithCause(errors.Warning("fns: endpoints merge member nodes failed")).WithCause(mergeErr)
			return
		}
		e.fetchRegistrations()
	}
	// http listen
	httpListenErrCh := make(chan error, 1)
	go func(srv Transport, ch chan error) {
		listenErr := srv.ListenAndServe()
		if listenErr != nil {
			ch <- errors.Warning("fns: run application failed").WithCause(listenErr)
			close(ch)
		}
	}(e.transport, httpListenErrCh)
	select {
	case <-time.After(1 * time.Second):
		break
	case httpErr := <-httpListenErrCh:
		e.rt.status.flag.Off()
		err = errors.Warning("fns: endpoints listen failed").WithCause(httpErr)
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
			e.rt.status.flag.Off()
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
	if e.cluster != nil {
		_ = e.cluster.Leave(ctx)
	}
	if e.registrations != nil {
		e.registrations.Close()
	}
	e.transportHandlers.Close()
	_ = e.transport.Close()
	for _, ep := range e.deployed {
		ep.svc.Close()
	}
	e.autoMaxProcs.Reset()
}

func (e *Endpoints) fetchRegistrations() {
	go func(e *Endpoints) {
		timer := time.NewTimer(e.clusterNodeFetchInterval)
		for {
			stop := false
			select {
			case <-e.closeCh:
				stop = true
				break
			case <-timer.C:
				// todo use registations.lister
				nodes, nodesErr := listMembers(context.TODO(), e.cluster, e.rt.appId, e.rt.appName, e.rt.appVersion)
				if nodesErr != nil {
					if e.log.WarnEnabled() {
						e.log.Warn().Cause(nodesErr).Message("fns: endpoints get nodes from cluster failed")
					}
					break
				}
				mergeErr := e.registrations.MergeNodes(nodes)
				if mergeErr != nil {
					if e.log.WarnEnabled() {
						e.log.Warn().Cause(mergeErr).Message("fns: endpoints merge nodes failed")
					}
				}
				break
			}
			if stop {
				timer.Stop()
				break
			}
			timer.Reset(e.clusterNodeFetchInterval)
		}
		timer.Stop()
	}(e)
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
	// todo with runtime move into http server base context
	// todo fasthttp rewrite fasthttpadaptor.NewFastHTTPHandler(options.Handler), set ctx with runtime
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

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
	"github.com/aacfactory/fns/service/internal/commons/flags"
	"github.com/aacfactory/fns/service/internal/logger"
	"github.com/aacfactory/fns/service/internal/procs"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/fns/service/shared"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type EndpointsOptions struct {
	OpenApiVersion string
	AppId          string
	AppName        string
	AppVersion     versions.Version
	Http           Http
	HttpHandlers   []HttpHandler
	Config         configures.Config
}

func NewEndpoints(options EndpointsOptions) (v *Endpoints, err error) {
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
	var sharedStore shared.Store
	var sharedLockers shared.Lockers
	var barrier Barrier
	if config.Cluster != nil {
		// cluster >>>
		kind := strings.TrimSpace(config.Cluster.Kind)
		builder, hasBuilder := getClusterBuilder(kind)
		if !hasBuilder {
			err = errors.Warning("fns: create endpoints failed").WithCause(errors.Warning("kind of cluster is not found").WithMeta("kind", kind))
			return
		}
		cluster, err = builder(ClusterBuilderOptions{
			Config:     config.Cluster,
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
		if config.Cluster.Shared != nil {
			if config.Cluster.Shared.BarrierDisabled {
				barrier = defaultBarrier()
			} else {
				barrierTTL := 100 * time.Millisecond
				if config.Cluster.Shared.BarrierTTLMilliseconds > 0 {
					barrierTTL = time.Duration(config.Cluster.Shared.BarrierTTLMilliseconds) * time.Millisecond
				}
				barrier = clusterBarrier(sharedStore, sharedLockers, barrierTTL)
			}
		}
	} else {
		// shared store >>>
		sharedMemSizeStr := strings.TrimSpace(runtimeConfig.LocalSharedStoreCacheSize)
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
	handleTimeoutSeconds := config.Runtime.HandleTimeoutSeconds
	if handleTimeoutSeconds < 1 {
		handleTimeoutSeconds = 10
	}
	// endpoints
	v = &Endpoints{
		rt: &runtimes{
			appId:         options.AppId,
			appName:       options.AppName,
			appVersion:    options.AppVersion,
			running:       flags.New(false),
			log:           log,
			worker:        worker,
			discovery:     nil,
			barrier:       barrier,
			sharedLockers: sharedLockers,
			sharedStore:   sharedStore,
			signer:        secret.NewSigner(bytex.FromString(secretKey)),
		},
		config:        options.Config,
		autoMaxProcs:  goprocs,
		log:           log,
		handleTimeout: time.Duration(handleTimeoutSeconds) * time.Second,
		deployed:      make(map[string]*endpoint),
		deployedCh:    make(chan map[string]*endpoint, 1),
		registrations: nil,
		http:          options.Http,
		httpHandlers:  nil,
		cluster:       cluster,
		closeCh:       make(chan struct{}, 1),
	}
	v.rt.discovery = v
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
		Running:    v.rt.running,
	})
	if handlersErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(handlersErr)
		return
	}
	v.httpHandlers = handlers
	appendHandlerErr := handlers.Append(newServiceHandler(bytex.FromString(secretKey), v.cluster != nil, v.deployedCh, options.OpenApiVersion))
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
		}
	}
	var httpHandler http.Handler = handlers
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
	buildErr := v.http.Build(HttpOptions{
		Port:      httpConfig.Port,
		ServerTLS: serverTLS,
		ClientTLS: clientTLS,
		Handler:   httpHandler,
		Log:       v.rt.log.With("http", v.http.Name()),
		Options:   httpConfig.Options,
	})
	if buildErr != nil {
		err = errors.Warning("fns: create endpoints failed").WithCause(buildErr)
		return
	}
	// http <<<
	return
}

type Endpoints struct {
	log           logs.Logger
	rt            *runtimes
	autoMaxProcs  *procs.AutoMaxProcs
	handleTimeout time.Duration
	config        configures.Config
	deployed      map[string]*endpoint
	deployedCh    chan map[string]*endpoint
	registrations *Registrations
	http          Http
	httpHandlers  *HttpHandlers
	cluster       Cluster
	closeCh       chan struct{}
}

func (e *Endpoints) Log() (log logs.Logger) {
	log = e.rt.log
	return
}

func (e *Endpoints) Get(ctx context.Context, service string, options ...EndpointDiscoveryGetOption) (v Endpoint, has bool) {
	if service == "" {
		return
	}
	opt := &EndpointDiscoveryGetOptions{
		id:           "",
		native:       false,
		versionRange: []versions.Version{versions.Min(), versions.Max()},
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
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
		if has && e.rt.appVersion.Between(opt.versionRange[0], opt.versionRange[1]) {
			return
		}
		has = false
		if e.registrations != nil && CanAccessInternal(ctx) && !opt.native {
			v, has = e.registrations.Get(service, opt.versionRange[0], opt.versionRange[1])
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
		err = errors.Warning(fmt.Sprintf("fns: endpoints deploy %s service failed", name)).WithCause(buildErr)
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
	e.autoMaxProcs.Enable()
	e.deployedCh <- e.deployed
	close(e.deployedCh)
	// cluster join
	if e.cluster != nil {
		joinErr := e.cluster.Join(ctx)
		if joinErr != nil {
			err = errors.Warning("fns: endpoints listen failed").WithCause(joinErr)
			return
		}
		// registrations
		e.registrations = &Registrations{
			id:      e.rt.appId,
			locker:  sync.Mutex{},
			values:  sync.Map{},
			dialer:  e.http,
			signer:  e.rt.signer,
			worker:  e.rt.worker,
			timeout: e.handleTimeout,
		}
		e.fetchRegistrations()
	}
	// http listen
	httpListenErrCh := make(chan error, 1)
	go func(srv Http, ch chan error) {
		listenErr := srv.ListenAndServe()
		if listenErr != nil {
			ch <- errors.Warning("fns: run application failed").WithCause(listenErr)
			close(ch)
		}
	}(e.http, httpListenErrCh)
	select {
	case <-time.After(1 * time.Second):
		break
	case httpErr := <-httpListenErrCh:
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
		}(withRuntime(context.TODO(), e.rt), ln, serviceListenErrCh)
	}
	if lns > 0 {
		select {
		case serviceListenErr := <-serviceListenErrCh:
			atomic.AddInt64(&closed, 1)
			err = errors.Warning("fns: endpoints listen failed").WithCause(serviceListenErr)
			return
		case <-time.After(time.Duration(lns) * time.Second):
			break
		}
	}
	e.rt.running.On()
	return
}

func (e *Endpoints) Running() (ok bool) {
	ok = e.rt.running.IsOn()
	return
}

func (e *Endpoints) Close(ctx context.Context) {
	e.closeCh <- struct{}{}
	close(e.closeCh)
	e.rt.running.Off()
	if e.cluster != nil {
		_ = e.cluster.Leave(ctx)
	}
	if e.registrations != nil {
		e.registrations.Close()
	}
	e.httpHandlers.Close()
	_ = e.http.Close()
	for _, ep := range e.deployed {
		ep.svc.Close()
	}
	e.autoMaxProcs.Reset()
}

func (e *Endpoints) fetchRegistrations() {
	go func(e *Endpoints) {
		for {
			stop := false
			select {
			case <-e.closeCh:
				stop = true
				break
			case <-time.After(1 * time.Second):
				nodes, nodesErr := e.cluster.Nodes(context.TODO())
				if nodesErr != nil {
					if e.log.WarnEnabled() {
						e.log.Warn().Cause(nodesErr).Message("fns: cluster get nodes failed")
					}
					break
				}
				if nodes == nil || len(nodes) == 0 {
					break
				}
				sort.Sort(nodes)
				nodesLen := nodes.Len()
				existIds := e.registrations.Ids()
				existIdsLen := len(existIds)
				newNodes := make([]Node, 0, 1)
				diffNodeIds := make([]string, 0, 1)
				for _, node := range nodes {
					if existIdsLen != 0 && sort.SearchStrings(existIds, node.Id) < existIdsLen {
						continue
					}
					newNodes = append(newNodes, node)
				}
				if existIdsLen > 0 {
					for _, id := range existIds {
						exist := sort.Search(nodesLen, func(i int) bool {
							return nodes[i].Id == id
						}) < nodesLen
						if exist {
							continue
						}
						diffNodeIds = append(diffNodeIds, id)
					}
				}
				if len(diffNodeIds) > 0 {
					for _, id := range diffNodeIds {
						e.registrations.Remove(id)
					}
				}
				if len(newNodes) > 0 {
					for _, node := range newNodes {
						fetchErr := e.registrations.AddNode(node)
						if fetchErr != nil {
							if e.log.WarnEnabled() {
								e.log.Warn().Cause(fetchErr).With("node_name", node.Name).With("node_id", node.Id).Message("fns: cluster fetch registration from node failed")
							}
						}
					}
				}
				break
			}
			if stop {
				return
			}
		}
	}(e)
	return
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

func (e *endpoint) Document() (document Document) {
	document = e.svc.Document()
	return
}

func (e *endpoint) Request(ctx context.Context, r Request) (result Result) {
	ctx = withRuntime(ctx, e.rt)
	ctx = withTracer(ctx, r.Id())
	ctx = withRequest(ctx, r)
	fr := NewResult()
	task := e.acquire()
	task.begin(r, fr)
	if !e.rt.worker.Dispatch(ctx, task) {
		serviceName, fnName := r.Fn()
		if ctx.Err() != nil {
			fr.Failed(errors.Timeout("fns: workers handle timeout").WithMeta("fns", "timeout").WithMeta("service", serviceName).WithMeta("fn", fnName))
		} else {
			fr.Failed(errors.NotAcceptable("fns: service is overload").WithMeta("fns", "overload").WithMeta("service", serviceName).WithMeta("fn", fnName))
		}
		e.release(task)
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

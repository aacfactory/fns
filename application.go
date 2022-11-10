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

package fns

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/internal/configure"
	"github.com/aacfactory/fns/internal/logger"
	"github.com/aacfactory/fns/internal/procs"
	"github.com/aacfactory/fns/listeners"
	"github.com/aacfactory/fns/server"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Application interface {
	Log() (log logs.Logger)
	Deploy(service ...service.Service) (err error)
	Run() (err error)
	RunWithHooks(ctx context.Context, hook ...Hook) (err error)
	Execute(ctx context.Context, serviceName string, fn string, argument interface{}, options ...ExecuteOption) (result interface{}, err errors.CodeError)
	Sync() (err error)
	Stop()
}

// +-------------------------------------------------------------------------------------------------------------------+

func New(options ...Option) (app Application) {
	opt := defaultOptions
	if options != nil {
		for _, option := range options {
			optErr := option(opt)
			if optErr != nil {
				panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(optErr)))
				return
			}
		}
	}
	// app
	appId := ""
	appVersion := opt.version
	// config
	configRetriever, configRetrieverErr := configures.NewRetriever(opt.configRetrieverOption)
	if configRetrieverErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed for invalid config retriever").WithCause(configRetrieverErr)))
		return
	}
	configRaw, configGetErr := configRetriever.Get()
	if configGetErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, get config via retriever failed").WithCause(configGetErr)))
		return
	}
	config := configure.Config{}
	decodeConfigErr := configRaw.As(&config)
	if decodeConfigErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, decode config failed").WithCause(decodeConfigErr)))
		return
	}
	appId = strings.TrimSpace(config.AppId)
	name := strings.TrimSpace(config.Name)
	if name == "" {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, name is undefined in config")))
		return
	}
	// running
	running := commons.NewSafeFlag(false)
	// log
	logOptions := logger.LogOptions{
		Name: name,
	}
	if config.Log != nil {
		logOptions.Color = config.Log.Color
		logOptions.Formatter = config.Log.Formatter
		logOptions.Level = config.Log.Level
	}
	log, logErr := logger.NewLog(logOptions)
	if logErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create logger failed").WithCause(logErr)))
		return
	}
	// extra listener
	extraListeners := opt.extraListeners

	// http server options
	httpOptions := server.HttpOptions{
		Port:      80,
		ServerTLS: nil,
		ClientTLS: nil,
		Handler:   nil,
		Log:       log,
		Raw:       nil,
	}
	if config.Server != nil {
		var serverTLS *tls.Config
		var clientTLS *tls.Config
		if config.Server.TLS != nil {
			var tlsErr error
			serverTLS, clientTLS, tlsErr = config.Server.TLS.Config()
			if tlsErr != nil {
				panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, load tls failed").WithCause(tlsErr)))
				return
			}
		}
		httpOptions.ServerTLS = serverTLS
		httpOptions.ClientTLS = clientTLS
		port := config.Server.Port
		if port == 0 {
			if serverTLS == nil {
				port = 80
			} else {
				port = 443
			}
		}
		if port < 1 || port > 65535 {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(fmt.Errorf("port is invalid, port must great than 1024 or less than 65536"))))
			return
		}
		httpOptions.Port = port
		if config.Server.Options == nil {
			config.Server.Options = []byte("{}")
		}
		httpOptions.Raw = json.NewObjectFromBytes(config.Server.Options)
	}

	// cluster
	var clusterManager *cluster.Manager
	if config.Cluster == nil {
		if appId == "" {
			appId = uid.UID()
		}
	} else {
		clientBuilder := opt.clientBuilder
		if clientBuilder == nil {
			clientBuilder = cluster.FastHttpClientBuilder
		}
		clusterManagerOptions := cluster.ManagerOptions{
			Log:               log,
			AppId:             appId,
			Port:              httpOptions.Port,
			Config:            config.Cluster,
			ClientHttps:       httpOptions.ServerTLS != nil,
			ClientTLS:         httpOptions.ClientTLS,
			ClientBuilder:     clientBuilder,
			DevMode:           config.Cluster.DevMode,
			NodesProxyAddress: config.Cluster.NodesProxyAddress,
		}
		var clusterManagerErr error
		clusterManager, clusterManagerErr = cluster.NewManager(clusterManagerOptions)
		if clusterManagerErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create cluster failed").WithCause(clusterManagerErr)))
			return
		}
		if appId == "" {
			appId = clusterManager.Node().Id()
		}
	}

	// procs
	goprocs := procs.New(procs.Options{
		Log: log,
		Min: opt.autoMaxProcsMin,
		Max: opt.autoMaxProcsMax,
	})

	// endpoints
	serviceMaxWorkers := 0
	serviceMaxIdleWorkerSeconds := 0
	serviceHandleTimeoutSeconds := 0
	if config.Runtime != nil {
		serviceMaxWorkers = config.Runtime.MaxWorkers
		if serviceMaxWorkers < 1 {
			serviceMaxWorkers = 0
		}
		serviceMaxIdleWorkerSeconds = config.Runtime.WorkerMaxIdleSeconds
		if serviceMaxIdleWorkerSeconds < 1 {
			serviceMaxIdleWorkerSeconds = 10
		}
		serviceHandleTimeoutSeconds = config.Runtime.HandleTimeoutSeconds
		if serviceHandleTimeoutSeconds < 1 {
			serviceHandleTimeoutSeconds = 10
		}
	}
	var discovery service.EndpointDiscovery
	if clusterManager != nil {
		discovery = clusterManager.Registrations()
	}
	endpoints := service.NewEndpoints(service.EndpointsOptions{
		AppId:                 appId,
		Running:               running,
		Log:                   log,
		MaxWorkers:            serviceMaxWorkers,
		MaxIdleWorkerDuration: time.Duration(serviceMaxIdleWorkerSeconds) * time.Second,
		HandleTimeout:         time.Duration(serviceHandleTimeoutSeconds) * time.Second,
		Barrier:               opt.barrier,
		Discovery:             discovery,
	})

	// http handler
	httpHandlers := server.NewHandlers(&server.HandlerOptions{
		Log:       log,
		Config:    configRaw,
		Endpoints: endpoints,
	})
	appendCorsErr := httpHandlers.Append(server.NewCorsHandler())
	if appendCorsErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(decodeConfigErr)))
		return
	}
	appendHealthErr := httpHandlers.Append(server.NewHealthHandler(server.HealthHandlerOptions{
		AppId:   appId,
		AppName: name,
		Version: appVersion,
		Running: running,
	}))
	if appendHealthErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(appendHealthErr)))
		return
	}
	if config.Server.Websocket != nil {
		appendWebsocketErr := httpHandlers.Append(server.NewWebsocketHandler())
		if appendWebsocketErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(appendWebsocketErr)))
			return
		}
	}
	appendDocumentErr := httpHandlers.Append(server.NewDocumentHandler(server.DocumentHandlerOptions{
		Version: appVersion,
	}))
	if appendDocumentErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(appendDocumentErr)))
		return
	}

	if len(opt.serverHandlers) > 0 {
		for _, handler := range opt.serverHandlers {
			if handler == nil {
				continue
			}
			appendErr := httpHandlers.Append(handler)
			if appendErr != nil {
				panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(appendErr)))
				return
			}
		}
	}
	if clusterManager != nil {
		_ = httpHandlers.Append(cluster.NewHandler(cluster.HandlerOptions{
			Log:       log,
			Endpoints: endpoints,
			Cluster:   clusterManager,
		}))
	}
	appendServiceErr := httpHandlers.Append(server.NewServiceHandler())
	if appendServiceErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(appendServiceErr)))
		return
	}

	// http server
	httpServer := opt.server
	if httpServer == nil {
		httpServer = &server.FastHttp{}
	}
	httpOptions.Handler = httpHandlers
	serverBuildErr := httpServer.Build(httpOptions)
	if serverBuildErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create http server failed").WithCause(serverBuildErr)))
		return
	}
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh,
		syscall.SIGINT,
		syscall.SIGKILL,
		syscall.SIGQUIT,
		syscall.SIGABRT,
		syscall.SIGTERM,
	)
	app = &application{
		log:             log,
		running:         running,
		autoMaxProcs:    goprocs,
		config:          configRaw,
		clusterManager:  clusterManager,
		endpoints:       endpoints,
		http:            httpServer,
		httpHandlers:    httpHandlers,
		extraListeners:  extraListeners,
		shutdownTimeout: opt.shutdownTimeout,
		signalCh:        signalCh,
		synced:          false,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type application struct {
	log             logs.Logger
	running         *commons.SafeFlag
	autoMaxProcs    *procs.AutoMaxProcs
	config          configures.Config
	clusterManager  *cluster.Manager
	endpoints       service.Endpoints
	http            server.Http
	httpHandlers    *server.Handlers
	extraListeners  []listeners.Listener
	shutdownTimeout time.Duration
	signalCh        chan os.Signal
	synced          bool
}

func (app *application) Log() (log logs.Logger) {
	log = app.log.With("fns", "application")
	return
}

func (app *application) Deploy(services ...service.Service) (err error) {
	if services == nil || len(services) == 0 {
		err = errors.Warning("fns: no services deployed")
		return
	}
	for _, svc := range services {
		if svc == nil {
			err = errors.Warning("fns: deploy service failed for one of services is nil")
			return
		}
		name := strings.TrimSpace(svc.Name())
		svcConfig, hasConfig := app.config.Node(name)
		if !hasConfig {
			svcConfig, _ = configures.NewJsonConfig([]byte("{}"))
		}
		buildErr := svc.Build(service.Options{
			Log:    app.log.With("fns", "service").With("service", name),
			Config: svcConfig,
		})
		if buildErr != nil {
			err = errors.Warning(fmt.Sprintf("fns: deploy %s service failed", name)).WithCause(buildErr)
			return
		}
		app.endpoints.Mount(svc)
		if app.clusterManager != nil {
			lns, isLn := svc.(service.Listenable)
			if !isLn {
				app.clusterManager.Node().AppendService(svc.Name(), svc.Internal())
			} else {
				if lns.Sharing() {
					app.clusterManager.Node().AppendService(svc.Name(), svc.Internal())
				}
			}
		}
	}
	return
}

func (app *application) Run() (err error) {
	if app.running.IsOn() {
		err = errors.Warning("fns: application is running")
		return
	}
	// goprocs
	app.autoMaxProcs.Enable()
	defer func(err error) {
		if err != nil {
			app.autoMaxProcs.Reset()
		}
	}(err)
	// extra listeners
	if app.extraListeners != nil && len(app.extraListeners) > 0 {
		for _, ln := range app.extraListeners {
			lnName := strings.TrimSpace(ln.Name())
			lnConfig, hasConfig := app.config.Node(lnName)
			if !hasConfig {
				lnConfig, _ = configures.NewJsonConfig([]byte("{}"))
			}
			lnOpt := listeners.ListenerOptions{
				Log:    app.log.With("extra_listener", lnName),
				Config: lnConfig,
			}
			lnCtx := app.endpoints.SetupContext(context.TODO())
			extraListenCh := make(chan error, 1)
			go func(ctx context.Context, ln listeners.Listener, opt *listeners.ListenerOptions, ch chan error) {
				listenErr := ln.Listen(lnCtx, *opt)
				if listenErr != nil {
					ch <- errors.Warning("fns: run application failed").WithCause(listenErr)
					close(ch)
				}
			}(lnCtx, ln, &lnOpt, extraListenCh)
			select {
			case <-time.After(1 * time.Second):
				break
			case httpErr := <-extraListenCh:
				err = httpErr
				return
			}
			inboundChannels := ln.OutboundChannels()
			if inboundChannels != nil {
				app.endpoints.RegisterOutboundChannels(lnName, inboundChannels)
			}
		}
	}
	// http start
	httpListenCh := make(chan error, 1)
	go func(srv server.Http, ch chan error) {
		listenErr := app.http.ListenAndServe()
		if listenErr != nil {
			ch <- errors.Warning("fns: run application failed").WithCause(listenErr)
			close(ch)
		}
	}(app.http, httpListenCh)
	select {
	case <-time.After(1 * time.Second):
		break
	case httpErr := <-httpListenCh:
		err = httpErr
		return
	}
	// cluster publish
	if app.clusterManager != nil {
		app.clusterManager.Join()
	}
	// on
	app.running.On()
	return
}

func (app *application) RunWithHooks(ctx context.Context, hooks ...Hook) (err error) {
	runErr := app.Run()
	if runErr != nil {
		err = runErr
		return
	}
	if hooks == nil || len(hooks) == 0 {
		return
	}

	ctx = service.TODO(ctx, app.endpoints)
	r, requestErr := service.NewInternalRequest("fns", "hooks", nil)
	if requestErr != nil {
		err = errors.Warning("fns run with hooks failed").WithCause(requestErr)
		return
	}
	ctx = service.SetRequest(ctx, r)
	config, hasConfig := app.config.Node("hooks")
	if !hasConfig {
		config, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	for _, hook := range hooks {
		if hook == nil {
			continue
		}
		hookConfig, hasHookConfig := config.Node(hook.Name())
		if !hasHookConfig {
			hookConfig, _ = configures.NewJsonConfig([]byte{'{', '}'})
		}
		buildErr := hook.Build(&HookOptions{
			Log:    app.log.With("hook", hook.Name()),
			Config: hookConfig,
		})
		if buildErr != nil {
			err = errors.Warning("fns run with hooks failed").WithCause(buildErr)
			return
		}
		ctx = service.SetLog(ctx, app.log.With("hoot", hook.Name()))
		hookErr := hook.Handle(ctx)
		if hookErr != nil {
			err = errors.Warning("fns run with hooks failed").WithCause(hookErr)
			return
		}
	}
	return
}

func (app *application) Execute(ctx context.Context, serviceName string, fn string, argument interface{}, options ...ExecuteOption) (result interface{}, err errors.CodeError) {
	ctx = service.TODO(ctx, app.endpoints)
	r, requestErr := service.NewInternalRequest(serviceName, fn, argument)
	if requestErr != nil {
		err = errors.Warning("fns execute failed").WithCause(requestErr).WithMeta("service", serviceName).WithMeta("fn", fn)
		return
	}
	opt := &ExecuteOptions{}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			optErr := option(opt)
			if optErr != nil {
				err = errors.Warning("fns execute failed").WithCause(optErr).WithMeta("service", serviceName).WithMeta("fn", fn)
				return
			}
		}
	}
	if opt.user != nil {
		r.SetUser(opt.user.Id(), opt.user.Attributes())
	}

	ctx = service.SetRequest(ctx, r)
	result, err = app.endpoints.Handle(ctx, r)
	return
}

func (app *application) Sync() (err error) {
	if app.synced {
		return
	}
	if app.running.IsOff() {
		err = errors.Warning("fns: application is not running")
		return
	}
	app.synced = true
	<-app.signalCh
	stopped := make(chan struct{}, 1)
	go app.stop(stopped)
	select {
	case <-time.After(app.shutdownTimeout):
		err = errors.Warning("fns: stop application timeout")
		break
	case <-stopped:
		if app.log.DebugEnabled() {
			app.log.Debug().Message("fns: stop application succeed")
		}
		break
	}
	return
}

func (app *application) stop(ch chan struct{}) {
	defer app.autoMaxProcs.Reset()
	// off
	app.running.Off()
	// cluster leave
	if app.clusterManager != nil {
		app.clusterManager.Leave()
	}
	// http
	app.httpHandlers.Close()
	httpCloseErr := app.http.Close()
	if httpCloseErr != nil {
		if app.log.WarnEnabled() {
			app.log.Warn().Cause(httpCloseErr).Message("fns: an error occurred in the stop application")
		}
	}
	// extra listeners
	if app.extraListeners != nil && len(app.extraListeners) > 0 {
		for _, listener := range app.extraListeners {
			lnCloseErr := listener.Close()
			if lnCloseErr != nil {
				if app.log.WarnEnabled() {
					app.log.Warn().Cause(lnCloseErr).Message("fns: an error occurred in the stop application")
				}
			}
		}
	}
	// endpoints
	app.endpoints.Close()
	ch <- struct{}{}
	close(ch)
	return
}

func (app *application) Stop() {
	if app.running.IsOff() {
		return
	}
	if !app.synced {
		go func(app *application) {
			_ = app.Sync()
		}(app)
		time.Sleep(1 * time.Second)
	}
	app.signalCh <- os.Kill
}

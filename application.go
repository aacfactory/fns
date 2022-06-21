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
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/internal/configuare"
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
	Sync() (err error)
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
	configRetriever, configRetrieverErr := configuares.NewRetriever(opt.configRetrieverOption)
	if configRetrieverErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed for invalid config retriever").WithCause(configRetrieverErr)))
		return
	}
	configRaw, configGetErr := configRetriever.Get()
	if configGetErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, get config via retriever failed").WithCause(configGetErr)))
		return
	}
	config := configuare.Config{}
	decodeConfigErr := configRaw.As(&config)
	if decodeConfigErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, decode config failed").WithCause(decodeConfigErr)))
		return
	}
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
		appId = uid.UID()
	} else {
		clientBuilder := opt.clientBuilder
		if clientBuilder == nil {
			clientBuilder = cluster.FastHttpClientBuilder
		}
		clientConfig := config.Cluster.Client
		maxIdleConnSeconds := clientConfig.MaxIdleConnSeconds
		if maxIdleConnSeconds < 1 {
			maxIdleConnSeconds = 10
		}
		maxConnsPerHost := clientConfig.MaxConnsPerHost
		if maxConnsPerHost < 1 {
			maxConnsPerHost = 0
		}
		maxIdleConnsPerHost := clientConfig.MaxIdleConnsPerHost
		if maxIdleConnsPerHost < 1 {
			maxIdleConnsPerHost = 0
		}
		requestTimeoutSeconds := clientConfig.RequestTimeoutSeconds
		if requestTimeoutSeconds < 1 {
			requestTimeoutSeconds = 2
		}
		client, clientErr := clientBuilder(cluster.ClientOptions{
			Log:                 log.With("cluster", "client"),
			Https:               httpOptions.ServerTLS != nil,
			TLS:                 httpOptions.ClientTLS,
			MaxIdleConnDuration: time.Duration(maxIdleConnSeconds) * time.Second,
			MaxConnsPerHost:     maxConnsPerHost,
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			RequestTimeout:      time.Duration(requestTimeoutSeconds) * time.Second,
		})
		if clientErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create cluster client failed").WithCause(clientErr)))
			return
		}
		clusterManagerOptions := cluster.ManagerOptions{
			Log:               log,
			Port:              httpOptions.Port,
			Kind:              config.Cluster.Kind,
			Options:           config.Cluster.Options,
			Client:            client,
			DevMode:           config.Cluster.DevMode,
			NodesProxyAddress: config.Cluster.NodesProxyAddress,
		}
		var clusterManagerErr error
		clusterManager, clusterManagerErr = cluster.NewManager(clusterManagerOptions)
		if clusterManagerErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create cluster failed").WithCause(clusterManagerErr)))
			return
		}
		appId = clusterManager.Node().Id()
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
	if config.Server != nil {
		serviceMaxWorkers = config.Service.MaxWorkers
		if serviceMaxWorkers < 1 {
			serviceMaxWorkers = 0
		}
		serviceMaxIdleWorkerSeconds = config.Service.WorkerMaxIdleSeconds
		if serviceMaxIdleWorkerSeconds < 1 {
			serviceMaxIdleWorkerSeconds = 10
		}
		serviceHandleTimeoutSeconds = config.Service.HandleTimeoutSeconds
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
	httpHandlers := server.NewHandlers()
	corsOptions := server.CorsHandlerOptions{
		Customized:       false,
		AllowedOrigins:   nil,
		AllowedHeaders:   nil,
		ExposedHeaders:   nil,
		AllowCredentials: false,
		MaxAge:           0,
	}
	if config.Server.Cors != nil {
		corsOptions = server.CorsHandlerOptions{
			Customized:       true,
			AllowedOrigins:   config.Server.Cors.AllowedOrigins,
			AllowedHeaders:   config.Server.Cors.AllowedHeaders,
			ExposedHeaders:   config.Server.Cors.ExposedHeaders,
			AllowCredentials: config.Server.Cors.AllowCredentials,
			MaxAge:           config.Server.Cors.MaxAge,
		}
	}
	httpHandlers.Append(server.NewCorsHandler(corsOptions))
	httpHandlers.Append(server.NewHealthHandler(server.HealthHandlerOptions{
		AppId:   appId,
		AppName: name,
		Version: appVersion,
		Running: running,
	}))
	docHandlerOptions := server.DocumentHandlerOptions{
		Log:       log,
		Version:   appVersion,
		Document:  nil,
		Endpoints: endpoints,
	}
	if config.OAS != nil {
		doc := server.Document{
			Title:       strings.TrimSpace(config.OAS.Title),
			Description: strings.TrimSpace(config.OAS.Description),
			Terms:       strings.TrimSpace(config.OAS.Terms),
			Contact:     nil,
			License:     nil,
			Addresses:   nil,
		}
		if config.OAS.Contact != nil {
			doc.Contact = &server.Contact{
				Name:  strings.TrimSpace(config.OAS.Contact.Name),
				Url:   strings.TrimSpace(config.OAS.Contact.Url),
				Email: strings.TrimSpace(config.OAS.Contact.Email),
			}
		}
		if config.OAS.License != nil {
			doc.License = &server.License{
				Name: strings.TrimSpace(config.OAS.License.Name),
				Url:  strings.TrimSpace(config.OAS.License.Url),
			}
		}
		if config.OAS.Servers != nil && len(config.OAS.Servers) > 0 {
			doc.Addresses = make([]server.Address, 0, 1)
			for _, oasServer := range config.OAS.Servers {
				doc.Addresses = append(doc.Addresses, server.Address{
					URL:         strings.TrimSpace(oasServer.URL),
					Description: strings.TrimSpace(oasServer.Description),
				})
			}
		}
		docHandlerOptions.Document = &doc
	}
	httpHandlers.Append(server.NewDocumentHandler(docHandlerOptions))
	if len(opt.serverInterceptorHandlers) > 0 {
		for _, handler := range opt.serverInterceptorHandlers {
			if handler == nil {
				continue
			}
			interceptorHandlerName := strings.TrimSpace(handler.Name())
			if interceptorHandlerName == "" {
				panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, one of interceptor handlers has no name")))
				return
			}
			interceptorHandlerLog := log.With("interceptor", interceptorHandlerName)
			var interceptorHandlerConfig configuares.Config
			var interceptorHandlerConfigGetErr error
			if config.Server.Interceptors != nil {
				interceptorHandlerConfigRaw, hasInterceptorHandlerConfig := config.Server.Interceptors[interceptorHandlerName]
				if hasInterceptorHandlerConfig {
					interceptorHandlerConfig, interceptorHandlerConfigGetErr = configuares.NewJsonConfig(interceptorHandlerConfigRaw)
				} else {
					interceptorHandlerConfig, interceptorHandlerConfigGetErr = configuares.NewJsonConfig([]byte{'{', '}'})
				}
			}
			if interceptorHandlerConfigGetErr != nil {
				panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, get interceptor handler config failed").WithCause(interceptorHandlerConfigGetErr).WithMeta("interceptor", interceptorHandlerName)))
				return
			}
			interceptorHBuildErr := handler.Build(server.InterceptorHandlerOptions{
				Log:    interceptorHandlerLog,
				Config: interceptorHandlerConfig,
			})
			if interceptorHBuildErr != nil {
				panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, build interceptor handler failed").WithCause(interceptorHBuildErr).WithMeta("interceptor", interceptorHandlerName)))
				return
			}
			httpHandlers.Append(handler)
		}
	}
	if clusterManager != nil {
		httpHandlers.Append(cluster.NewHandler(cluster.HandlerOptions{
			Log:       log,
			Endpoints: endpoints,
			Cluster:   clusterManager,
		}))
	}
	httpHandlers.Append(server.NewServiceHandler(server.ServiceHandlerOptions{
		Log:       log,
		Endpoints: endpoints,
	}))

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
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type application struct {
	log             logs.Logger
	running         *commons.SafeFlag
	autoMaxProcs    *procs.AutoMaxProcs
	config          configuares.Config
	clusterManager  *cluster.Manager
	endpoints       service.Endpoints
	http            server.Http
	httpHandlers    *server.Handlers
	extraListeners  []listeners.Listener
	shutdownTimeout time.Duration
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
			svcConfig, _ = configuares.NewJsonConfig([]byte("{}"))
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
				lnConfig, _ = configuares.NewJsonConfig([]byte("{}"))
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

func (app *application) Sync() (err error) {
	if app.running.IsOff() {
		err = errors.Warning("fns: application is not running")
		return
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGKILL,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)
	<-ch
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

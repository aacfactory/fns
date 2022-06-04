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
	sc "context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/fns/documents"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Application interface {
	Log() (log logs.Logger)
	Deploy(service ...Service) (err error)
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
				panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed for invalid options").WithCause(optErr)))
				return
			}
		}
	}
	// app
	appId := ""
	appAddress := ""
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
	config := Config{}
	decodeConfigErr := configRaw.As(&config)
	if decodeConfigErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, decode config failed").WithCause(decodeConfigErr)))
		return
	}
	// log
	log, logErr := newLog(config.Log)
	if logErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create logger failed").WithCause(logErr)))
		return
	}
	// tls
	ssl := false
	var servetTLS *tls.Config
	var clientTLS *tls.Config
	if config.TLS != nil {
		tlsOptions, hasTlsOptions := configRaw.Node("tls.options")
		if !hasTlsOptions {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create tls failed").WithCause(fmt.Errorf("fns: no tls options in config"))))
			return
		}
		servetTLS0, clientTLS0, tlsErr := config.TLS.Load(tlsOptions)
		if tlsErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, load tls failed").WithCause(tlsErr)))
			return
		}
		servetTLS = servetTLS0
		clientTLS = clientTLS0
		ssl = true
	}
	// port
	port := config.Port
	if port == 0 {
		if servetTLS == nil {
			port = 80
		} else {
			port = 443
		}
	}
	if port < 1024 || port > 65535 {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(fmt.Errorf("port is invalid, port must great than 1024 or less than 65536"))))
		return
	}
	// cluster
	var clusterManager *cluster.Manager
	if config.Cluster == nil {
		appId = UID()
		appIp := commons.GetGlobalUniCastIpFromHostname()
		if appIp == "" {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(fmt.Errorf("can not get ip from hostname, please set FNS_IP into system env"))))
			return
		}
		appAddress = fmt.Sprintf("%s:%d", appIp, port)
	} else {
		clusterManagerOptions := cluster.ManagerOptions{
			Log:           log,
			Port:          port,
			Config:        config.Cluster,
			ClientTLS:     clientTLS,
			ClientBuilder: opt.clientBuilder,
		}
		var clusterManagerErr error
		clusterManager, clusterManagerErr = cluster.NewManager(clusterManagerOptions)
		if clusterManagerErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create cluster failed").WithCause(clusterManagerErr)))
			return
		}
		appId = clusterManager.Node().Id
		appAddress = clusterManager.Node().Address
	}
	// running
	running := commons.NewSafeFlag(false)
	// env
	env := newEnvironments(appId, appAddress, opt.document.Version, running, configRaw, log)
	// procs
	goprocs := newPROCS(env, opt.procs)
	// documents
	document := opt.document

	// endpoints
	endpoints, endpointsErr := newEndpoints(env, serviceEndpointsOptions{
		workerMaxIdleTime: opt.workerMaxIdleTime,
		barrier:           opt.barrier,
		clusterManager:    clusterManager,
	})
	if endpointsErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create endpoints failed").WithCause(endpointsErr)))
		return
	}
	// runtime
	runtime := newServiceRuntime(env, endpoints, opt.validator)

	// todo auth
	// todo permissions

	// hooks
	hs, hooksErr := newHooks(env, opt.hooks)
	if hooksErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create hooks failed").WithCause(hooksErr)))
		return
	}
	// tracer
	tracerReporter := opt.tracerReporter
	tracerReporterErr := tracerReporter.Build(env)
	if tracerReporterErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create tracer reporter failed").WithCause(tracerReporterErr)))
		return
	}

	// http server
	httpServerHandler := newHttpHandler(env, httpHandlerOptions{
		env:                  env,
		document:             document,
		barrier:              opt.barrier,
		requestHandleTimeout: opt.handleRequestTimeout,
		runtime:              runtime,
		tracerReporter:       tracerReporter,
		hooks:                hs,
	})

	var httpServerHandlers http.Handler = httpServerHandler
	if clusterManager != nil {
		clusterHandler := cluster.NewHandler(clusterManager)
		httpServerHandlers = clusterHandler.Handler(httpServerHandlers)
	}
	httpHandlerWrapperBuildersLen := len(opt.httpHandlerWrapperBuilders)
	for i := httpHandlerWrapperBuildersLen; i > 0; i-- {
		wrapper, wrapperErr := opt.httpHandlerWrapperBuilders[i-1](env)
		if wrapperErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create http handler wrapper failed").WithCause(wrapperErr)))
			return
		}
		httpServerHandlers = wrapper.Wrap(httpServerHandlers)
	}

	var serverOptionsRaw *json.Object
	if config.ServerOptions != nil && len(config.ServerOptions) > 2 {
		serverOptionsRaw = json.NewObjectFromBytes(config.ServerOptions)
	} else {
		serverOptionsRaw = json.NewObject()
	}
	httpServer, httpServerErr := opt.serverBuilder(HttpServerOptions{
		Port:    port,
		TLS:     servetTLS,
		Handler: httpServerHandlers,
		Log:     log.With("fns", "http"),
		raw:     serverOptionsRaw,
	})
	// http server
	if httpServerErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create http server failed").WithCause(httpServerErr)))
		return
	}

	app = &application{
		env:             env,
		running:         running,
		log:             env.log.With("system", "application"),
		goprocs:         goprocs,
		document:        document,
		services:        make(map[string]Service),
		endpoints:       endpoints,
		clusterManager:  clusterManager,
		https:           ssl,
		http:            httpServer,
		httpHandler:     httpServerHandler,
		tracerReporter:  tracerReporter,
		hooks:           hs,
		shutdownTimeout: opt.shutdownTimeout,
	}

	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type application struct {
	env             Environments
	running         *commons.SafeFlag
	log             logs.Logger
	goprocs         *procs
	document        *documents.Application
	services        map[string]Service
	endpoints       *serviceEndpoints
	clusterManager  *cluster.Manager
	https           bool
	http            HttpServer
	httpHandler     *httpHandler
	tracerReporter  TracerReporter
	hooks           *hooks
	shutdownTimeout time.Duration
}

func (app *application) Log() (log logs.Logger) {
	log = app.log
	return
}

func (app *application) Deploy(services ...Service) (err error) {
	if services == nil || len(services) == 0 {
		err = errors.Warning("fns: no services deployed")
		return
	}
	for _, service := range services {
		if service == nil {
			err = errors.Warning("fns: deploy service failed for it is nil")
			return
		}
		name := strings.TrimSpace(service.Name())
		_, has := app.services[name]
		if has {
			err = errors.Warning(fmt.Sprintf("fns: %s service has been deployed", name))
			return
		}
		buildErr := service.Build(app.env)
		if buildErr != nil {
			err = errors.Warning(fmt.Sprintf("fns: deploy %s service failed", name)).WithCause(buildErr)
			return
		}
		app.services[name] = service
		// endpoints
		app.endpoints.mount(service)
		// documents
		if !service.Internal() {
			doc := service.Document()
			if doc != nil {
				app.document.AddService(name, doc)
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
	app.goprocs.enable()
	defer func(err error) {
		if err != nil {
			app.goprocs.reset()
		}
	}(err)
	// endpoints
	endpointsErr := app.endpoints.start()
	if endpointsErr != nil {
		err = errors.Warning("fns: run application failed").WithCause(endpointsErr)
		return
	}
	// http start
	httpListenCh := make(chan error, 1)
	go func(srv HttpServer, ch chan error) {
		listenErr := app.http.ListenAndServe()
		if listenErr != nil {
			ch <- errors.Warning("fns: run application failed").WithCause(listenErr)
		}
	}(app.http, httpListenCh)
	select {
	case <-time.After(1 * time.Second):
		close(httpListenCh)
		break
	case httpErr := <-httpListenCh:
		close(httpListenCh)
		err = httpErr
		return
	}
	// hooks
	app.hooks.start()
	// cluster publish
	if app.clusterManager != nil {
		for _, service := range app.services {
			if service.Internal() {
				app.clusterManager.Node().AppendInternalService(service.Name())
			} else {
				app.clusterManager.Node().AppendService(service.Name())
			}
		}
		app.clusterManager.Join(sc.TODO())
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
	ctx, cancel := sc.WithTimeout(sc.TODO(), app.shutdownTimeout)
	go app.stop(ctx, stopped)
	select {
	case <-ctx.Done():
		err = errors.Warning("fns: stop application timeout")
		break
	case <-stopped:
		break
	}
	cancel()
	return
}

func (app *application) stop(ctx sc.Context, ch chan struct{}) {
	defer func(reset *procs) {
		reset.reset()
	}(app.goprocs)
	// off
	app.running.Off()
	// cluster leave
	if app.clusterManager != nil {
		app.clusterManager.Leave(ctx)
	}
	// http
	httpHandlerCloseErr := app.httpHandler.Close()
	if httpHandlerCloseErr != nil {
		if app.log.WarnEnabled() {
			app.log.Warn().Cause(httpHandlerCloseErr).Message("fns: stop application failed")
		}
	}
	httpCloseErr := app.http.Close()
	if httpCloseErr != nil {
		if app.log.WarnEnabled() {
			app.log.Warn().Cause(httpCloseErr).Message("fns: stop application failed")
		}
	}
	// tracerReporter
	tracerReporterCloseErr := app.tracerReporter.Close()
	if tracerReporterCloseErr != nil {
		if app.log.WarnEnabled() {
			app.log.Warn().Cause(tracerReporterCloseErr).Message("fns: stop application failed")
		}
	}
	// hooks
	hooksCloseErr := app.hooks.close()
	if hooksCloseErr != nil {
		if app.log.WarnEnabled() {
			app.log.Warn().Cause(hooksCloseErr).Message("fns: stop application failed")
		}
	}
	// endpoints
	endpointsCloseErr := app.endpoints.close()
	if endpointsCloseErr != nil {
		if app.log.WarnEnabled() {
			app.log.Warn().Cause(endpointsCloseErr).Message("fns: stop application failed")
		}
	}
	// services
	for _, service := range app.services {
		serviceErr := service.Shutdown(ctx)
		if serviceErr != nil {
			if app.log.WarnEnabled() {
				app.log.Warn().Cause(serviceErr).Message("fns: stop application failed")
			}
		}
	}
	ch <- struct{}{}
	close(ch)
	return
}

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
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/logs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
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
	// running
	running := commons.NewSafeFlag(false)
	// env
	env := newEnvironments(UID(), opt.documents.Version, running, configRaw, log)
	// procs
	goprocs := newPROCS(env, opt.procs)
	// documents
	document := opt.documents
	documentsErr := document.setURL(env)
	if documentsErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create document failed").WithCause(documentsErr)))
		return
	}
	// discovery
	discovery, createDiscoveryErr := newDiscovery(env)
	if createDiscoveryErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create discovery failed").WithCause(createDiscoveryErr)))
		return
	}
	// endpoints
	endpoints, endpointsErr := newEndpoints(env, serviceEndpointsOptions{
		concurrency:       opt.concurrency,
		workerMaxIdleTime: opt.workerMaxIdleTime,
		barrier:           opt.barrier,
		discovery:         discovery,
	})
	if endpointsErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create endpoints failed").WithCause(endpointsErr)))
		return
	}
	// runtime
	runtime := newServiceRuntime(env, endpoints, opt.validator)

	// websocket
	websocketDiscovery := opt.websocketDiscovery
	websocketDiscoveryErr := websocketDiscovery.Build(env)
	if websocketDiscoveryErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create websocket discovery failed").WithCause(websocketDiscoveryErr)))
		return
	}
	RegisterEmbedService(&websocketService{
		discovery: websocketDiscovery,
	})
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

	// http handler
	httpServerHandler, httpServerHandlerErr := newHttpHandler(env, httpHandlerOptions{
		env:                  env,
		documents:            document,
		barrier:              opt.barrier,
		requestHandleTimeout: opt.serviceRequestTimeout,
		websocketDiscovery:   websocketDiscovery,
		runtime:              runtime,
		tracerReporter:       tracerReporter,
		hooks:                hs,
	})
	if httpServerHandlerErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create http handler failed").WithCause(httpServerHandlerErr)))
		return
	}

	var httpServerHandlers http.Handler = httpServerHandler
	for _, wrapper := range opt.httpHandlerWrappers {
		wrapperErr := wrapper.Build(env)
		if wrapperErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create http handler wrapper failed").WithCause(wrapperErr)))
			return
		}
		httpServerHandlers = wrapper.Handler(httpServerHandlers)
	}
	// http server
	httpServer := opt.server
	httpServerErr := httpServer.Build(env, httpServerHandlers)
	if httpServerErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create http server failed").WithCause(httpServerErr)))
		return
	}

	app = &application{
		env:                   env,
		running:               running,
		log:                   env.log.With("system", "application"),
		goprocs:               goprocs,
		documents:             document,
		services:              make(map[string]Service),
		endpoints:             endpoints,
		discovery:             discovery,
		registrations:         make([]*Registration, 0, 1),
		registrationClientTLS: getRegistrationClientTLS(env),
		http:                  httpServer,
		httpHandler:           httpServerHandler,
		httpVersion:           opt.httpVersion,
		tracerReporter:        tracerReporter,
		hooks:                 hs,
		shutdownTimeout:       opt.shutdownTimeout,
	}

	for _, embed := range embedServices {
		deployErr := app.Deploy(embed)
		if deployErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create embed service failed").WithCause(deployErr)))
			return
		}
	}

	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type application struct {
	env                   Environments
	running               *commons.SafeFlag
	log                   logs.Logger
	goprocs               *procs
	documents             *Documents
	services              map[string]Service
	endpoints             *serviceEndpoints
	discovery             Discovery
	registrations         []*Registration
	registrationClientTLS RegistrationClientTLS
	http                  HttpServer
	httpHandler           *httpHandler
	httpVersion           string
	tracerReporter        TracerReporter
	hooks                 *hooks
	shutdownTimeout       time.Duration
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
				app.documents.addServiceDocument(name, doc)
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
		listenErr := app.http.Listen()
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
	// discovery publish
	if app.discovery != nil {
		address := app.documents.host
		httpVersion := app.httpVersion
		for _, service := range app.services {
			app.registrations = append(app.registrations, &Registration{
				Id:          app.env.AppId(),
				Name:        strings.TrimSpace(service.Name()),
				Internal:    service.Internal(),
				Address:     address,
				HttpVersion: httpVersion,
				ClientTLS:   app.registrationClientTLS,
				once:        sync.Once{},
			})
		}
		registerErr := app.discovery.Register(app.registrations)
		if registerErr != nil {
			err = errors.Warning("fns: run application failed").WithCause(registerErr)
			return
		}
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
		os.Kill,
		syscall.SIGKILL,
		syscall.SIGTERM,
	)
	<-ch
	stopped := make(chan struct{}, 1)
	select {
	case <-time.After(app.shutdownTimeout):
		err = errors.Warning("fns: stop application timeout")
		break
	case <-stopped:
		break
	}
	return
}

func (app *application) stop(ch chan struct{}) {
	defer func(reset *procs) {
		reset.reset()
	}(app.goprocs)
	// off
	app.running.Off()
	// discovery deregister
	if app.discovery != nil {
		deregisterErr := app.discovery.Deregister(app.registrations)
		if deregisterErr != nil {
			if app.log.WarnEnabled() {
				app.log.Warn().Cause(deregisterErr).Message("fns: stop application failed")
			}
		}
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
	// endpoints
	endpointsCloseErr := app.endpoints.close()
	if endpointsCloseErr != nil {
		if app.log.WarnEnabled() {
			app.log.Warn().Cause(endpointsCloseErr).Message("fns: stop application failed")
		}
	}
	// services
	for _, service := range app.services {
		serviceErr := service.Shutdown()
		if serviceErr != nil {
			if app.log.WarnEnabled() {
				app.log.Warn().Cause(serviceErr).Message("fns: stop application failed")
			}
		}
	}
	// discovery close
	if app.discovery != nil {
		closeErr := app.discovery.Close()
		if closeErr != nil {
			if app.log.WarnEnabled() {
				app.log.Warn().Cause(closeErr).Message("fns: stop application failed")
			}
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
	ch <- struct{}{}
	close(ch)
	return
}

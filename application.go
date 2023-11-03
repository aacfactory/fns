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
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/commons/procs"
	"github.com/aacfactory/fns/commons/switchs"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/configs"
	"github.com/aacfactory/fns/handlers"
	"github.com/aacfactory/fns/hooks"
	"github.com/aacfactory/fns/log"
	"github.com/aacfactory/fns/proxies"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/workers"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Application interface {
	Deploy(service ...services.Service) Application
	Run(ctx context.Context) Application
	Sync()
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
	appId := strings.TrimSpace(opt.id)
	if appId == "" {
		appId = uid.UID()
	}
	appName := opt.name
	appVersion := opt.version
	// status
	status := &switchs.Switch{}
	// config
	configRetriever, configRetrieverErr := configures.NewRetriever(opt.configRetrieverOption)
	if configRetrieverErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed for invalid config retriever").WithCause(configRetrieverErr)))
		return
	}
	configure, configureErr := configRetriever.Get()
	if configureErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, get config via retriever failed").WithCause(configureErr)))
		return
	}
	config := configs.Config{}
	configErr := configure.As(&config)
	if configErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, decode config failed").WithCause(configErr)))
		return
	}
	// log
	loggger, loggerErr := log.New(appName, config.Log)
	if loggerErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new log failed").WithCause(loggerErr)))
		return
	}

	// proc
	amp := procs.New(config.Procs.Min, config.Procs.Max)
	// worker
	workerOptions := make([]workers.Option, 0, 1)
	if config.Workers.Max > 0 {
		workerOptions = append(workerOptions, workers.MaxWorkers(config.Workers.Max))
	}
	if config.Workers.MaxIdleSeconds > 0 {
		workerOptions = append(workerOptions, workers.MaxIdleWorkerDuration(time.Duration(config.Workers.MaxIdleSeconds)*time.Second))
	}
	worker := workers.New(workerOptions...)

	// barrier
	var barrier *barriers.Barrier
	// cluster
	var registrations *clusters.Registrations
	var shared shareds.Shared
	if clusterConfig := config.Cluster; clusterConfig != nil {
		port, portErr := config.Transport.Port()
		if portErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new cluster failed").WithCause(portErr)))
			return
		}
		var registrationsErr error
		registrations, registrationsErr = clusters.New(clusters.Options{
			Id:      appId,
			Name:    appName,
			Version: appVersion,
			Port:    port,
			Log:     loggger.Logger,
			Dialer:  opt.transport,
			Config:  *clusterConfig,
		})
		if registrationsErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new cluster failed").WithCause(registrationsErr)))
			return
		}
		shared = registrations.Shared()
		if barrierConfig := clusterConfig.Barrier; barrierConfig != nil {
			barrier = barriers.Cluster(shared.Store(), barrierConfig.TTL, barrierConfig.Interval)
		} else {
			shared = shareds.Local()
		}
	} else {
		shared = shareds.Local()
		barrier = barriers.Standalone()
	}
	// endpoints
	endpoints := services.New(
		appId, appName, appVersion,
		loggger.Logger, config.Services, worker,
		registrations,
	)

	// runtime
	rt := runtime.New(
		appId, appName, appVersion,
		status, loggger.Logger, worker,
		endpoints, registrations,
		barrier, shared,
		opt.transport,
	)

	// transport >>>
	// middlewares
	middlewares := make([]transports.Middleware, 0, 1)
	middlewares = append(middlewares, handlers.NewApplicationMiddleware(rt))
	var corsMiddleware transports.Middleware
	for _, middleware := range opt.middlewares {
		if middleware.Name() == "cors" {
			corsMiddleware = middleware
			continue
		}
		middlewares = append(middlewares, middleware)
	}
	if corsMiddleware != nil {
		middlewares = append([]transports.Middleware{corsMiddleware}, middlewares...)
	}
	// handler
	mux := transports.NewMux()
	mux.Add(handlers.NewEndpointsHandler())
	mux.Add(handlers.NewHealthHandler())
	mux.Add(handlers.NewDocumentHandler())
	if registrations != nil {
		mux.Add(clusters.NewInternalHandler(appId, registrations.Signature()))
	}
	transport := opt.transport
	transportErr := transport.Construct(transports.Options{
		Log:         loggger.Logger,
		Config:      config.Transport,
		Middlewares: middlewares,
		Handler:     mux,
	})
	if transportErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new transport failed").WithCause(transportErr)))
		return
	}
	// transport <<<

	// proxy >>>
	var proxy proxies.Proxy
	if proxyOptions := opt.proxyOptions; len(proxyOptions) > 0 {
		var proxyErr error
		proxy, proxyErr = proxies.New(proxyOptions...)
		if proxyErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new proxy failed").WithCause(proxyErr)))
			return
		}
		constructErr := proxy.Construct(proxies.ProxyOptions{
			Log:     loggger.Logger.With("fns", "proxy"),
			Config:  config.Proxy,
			Runtime: rt,
		})
		if constructErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new proxy failed").WithCause(constructErr)))
			return
		}
	}
	// proxy <<<

	// hooks
	for _, hook := range opt.hooks {
		hookConfig, hookConfigErr := config.Hooks.Get(hook.Name())
		if hookConfigErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new hook failed").WithCause(hookConfigErr)))
			return
		}
		hookErr := hook.Construct(hooks.Options{
			Log:    loggger.Logger.With("hook", hook.Name()),
			Config: hookConfig,
		})
		if hookErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new hook failed").WithCause(hookErr)))
			return
		}
	}

	// signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh,
		syscall.SIGINT,
		syscall.SIGKILL,
		syscall.SIGQUIT,
		syscall.SIGABRT,
		syscall.SIGTERM,
	)
	app = &application{
		id:              appId,
		name:            appName,
		version:         appVersion,
		rt:              rt,
		status:          status,
		log:             loggger,
		config:          config,
		amp:             amp,
		worker:          worker,
		endpoints:       endpoints,
		registrations:   registrations,
		transport:       transport,
		proxy:           proxy,
		hooks:           opt.hooks,
		shutdownTimeout: opt.shutdownTimeout,
		synced:          false,
		signalCh:        signalCh,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type application struct {
	id              string
	name            string
	version         versions.Version
	rt              *runtime.Runtime
	status          *switchs.Switch
	log             *log.Logger
	config          configs.Config
	amp             *procs.AutoMaxProcs
	worker          workers.Workers
	endpoints       *services.Services
	registrations   *clusters.Registrations
	transport       transports.Transport
	proxy           proxies.Proxy
	hooks           []hooks.Hook
	shutdownTimeout time.Duration
	synced          bool
	signalCh        chan os.Signal
}

func (app *application) Deploy(s ...services.Service) Application {
	for _, service := range s {
		err := app.endpoints.Add(service)
		if err != nil {
			panic(fmt.Sprintf("%+v", errors.Warning("fns: deploy failed").WithCause(err)))
			return app
		}
		if app.registrations != nil {
			app.registrations.Add(service.Name(), service.Internal())
		}
	}
	return app
}

func (app *application) Run(ctx context.Context) Application {
	// transport
	trErrs := make(chan error, 1)
	go func(ctx context.Context, transport transports.Transport, errs chan error) {
		lnErr := transport.ListenAndServe()
		if lnErr != nil {
			errs <- lnErr
			close(errs)
		}
	}(ctx, app.transport, trErrs)
	select {
	case trErr := <-trErrs:
		panic(fmt.Sprintf("%+v", errors.Warning("fns: application run failed").WithCause(trErr)))
		return app
	case <-time.After(3 * time.Second):
		break
	}
	app.status.On()
	app.status.Confirm()
	// registrations
	if app.registrations != nil {
		watchErr := app.registrations.Watching(ctx)
		if watchErr != nil {
			app.shutdown()
			panic(fmt.Sprintf("%+v", errors.Warning("fns: application run failed").WithCause(watchErr)))
			return app
		}
	}
	// endpoints
	lnErr := app.endpoints.Listen(ctx)
	if lnErr != nil {
		app.shutdown()
		panic(fmt.Sprintf("%+v", errors.Warning("fns: application run failed").WithCause(lnErr)))
		return app
	}
	// proxy
	if app.proxy != nil {
		prErrs := make(chan error, 1)
		go func(ctx context.Context, proxy proxies.Proxy, errs chan error) {
			proxyErr := proxy.Run(ctx)
			if proxyErr != nil {
				errs <- proxyErr
				close(errs)
			}
		}(ctx, app.proxy, prErrs)
		select {
		case prErr := <-prErrs:
			app.shutdown()
			panic(fmt.Sprintf("%+v", errors.Warning("fns: application run failed").WithCause(prErr)))
			return app
		case <-time.After(3 * time.Second):
			break
		}
	}
	// hooks
	for _, hook := range app.hooks {
		app.worker.MustDispatch(runtime.With(ctx, app.rt), hook)
	}
	return app
}

func (app *application) Sync() {
	if app.synced {
		return
	}
	app.synced = true
	<-app.signalCh
	app.shutdown()
	return
}

func (app *application) shutdown() {
	timeout := app.shutdownTimeout
	if timeout < 1 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// status
	app.status.Off()
	app.status.Confirm()
	go func(ctx context.Context, app *application) {
		// cluster
		if app.registrations != nil {
			rc, rcc := context.WithCancel(ctx)
			app.registrations.Shutdown(rc)
			rcc()
		}
		// endpoints
		ec, ecc := context.WithCancel(ctx)
		app.endpoints.Shutdown(ec)
		ecc()
		// proxy
		if app.proxy != nil {
			pc, pcc := context.WithCancel(ctx)
			app.proxy.Shutdown(pc)
			pcc()
		}
		// transport
		tc, tcc := context.WithCancel(ctx)
		app.transport.Shutdown(tc)
		tcc()
		// log
		lc, lcc := context.WithCancel(ctx)
		app.log.Shutdown(lc)
		lcc()
	}(ctx, app)
	<-ctx.Done()
	cancel()
}

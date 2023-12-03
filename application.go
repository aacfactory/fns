/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package fns

import (
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
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/hooks"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/proxies"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/workers"
	"os"
	"os/signal"
	"strconv"
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
	logger, loggerErr := logs.New(appName, config.Log)
	if loggerErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new log failed").WithCause(loggerErr)))
		return
	}

	// proc
	amp := procs.New(config.Runtime.Procs.Min)
	// worker
	workerOptions := make([]workers.Option, 0, 1)
	if workersMax := config.Runtime.Workers.Max; workersMax > 0 {
		workerOptions = append(workerOptions, workers.MaxWorkers(workersMax))
	}
	if workersMaxIdleSeconds := config.Runtime.Workers.MaxIdleSeconds; workersMaxIdleSeconds > 0 {
		workerOptions = append(workerOptions, workers.MaxIdleWorkerDuration(time.Duration(workersMaxIdleSeconds)*time.Second))
	}
	worker := workers.New(workerOptions...)

	handlers := make([]transports.MuxHandler, 0, 1)

	var manager services.EndpointsManager

	local := services.New(appId, appVersion, logger.With("fns", "endpoints"), config.Services, worker)

	handlers = append(handlers, services.Handler(local))
	handlers = append(handlers, runtime.HealthHandler())

	// barrier
	var barrier barriers.Barrier
	// shared
	var shared shareds.Shared
	// cluster
	if clusterConfig := config.Cluster; clusterConfig.Name != "" {
		port, portErr := config.Transport.GetPort()
		if portErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(portErr)))
			return
		}
		var clusterHandlers []transports.MuxHandler
		var clusterErr error
		manager, shared, barrier, clusterHandlers, clusterErr = clusters.New(clusters.Options{
			Id:      appId,
			Version: appVersion,
			Port:    port,
			Log:     logger.With("fns", "cluster"),
			Worker:  worker,
			Local:   local,
			Dialer:  opt.transport,
			Config:  clusterConfig,
		})
		if clusterErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(clusterErr)))
			return
		}
		handlers = append(handlers, clusterHandlers...)
	} else {
		var sharedErr error
		shared, sharedErr = shareds.Local(logger.With("shared", "local"), config.Runtime.Shared)
		if sharedErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(sharedErr)))
			return
		}
		barrier = barriers.New()
		manager = local
	}

	// runtime
	rt := runtime.New(
		appId, appName, appVersion,
		status, logger, worker,
		manager,
		barrier, shared,
	)

	// builtins
	builtins := make([]services.Service, 0, 1)

	// transport >>>
	// middlewares
	middlewares := make([]transports.Middleware, 0, 1)
	middlewares = append(middlewares, runtime.Middleware(rt))
	var corsMiddleware transports.Middleware
	for _, middleware := range opt.middlewares {
		builtin, isBuiltin := middleware.(services.Middleware)
		if isBuiltin {
			builtins = append(builtins, builtin.Services()...)
		}
		if middleware.Name() == "cors" {
			corsMiddleware = middleware
			continue
		}
		middlewares = append(middlewares, middleware)
	}
	if corsMiddleware != nil {
		middlewares = append([]transports.Middleware{corsMiddleware}, middlewares...)
	}
	middleware, middlewareErr := transports.WaveMiddlewares(logger, config.Transport, middlewares)
	if middlewareErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new transport middleware failed").WithCause(middlewareErr)))
		return
	}
	// handler
	mux := transports.NewMux()
	handlers = append(handlers, opt.handlers...)
	for _, handler := range handlers {
		handlerConfig, handlerConfigErr := config.Transport.HandlerConfig(handler.Name())
		if handlerConfigErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new transport handler failed").WithCause(handlerConfigErr).WithMeta("handler", handler.Name())))
			return
		}
		handlerErr := handler.Construct(transports.MuxHandlerOptions{
			Log:    logger.With("handler", handler.Name()),
			Config: handlerConfig,
		})
		if handlerErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new transport handler failed").WithCause(handlerErr).WithMeta("handler", handler.Name())))
			return
		}
		mux.Add(handler)
		builtin, isBuiltin := handler.(services.MuxHandler)
		if isBuiltin {
			builtins = append(builtins, builtin.Services()...)
		}
	}
	transport := opt.transport
	transportErr := transport.Construct(transports.Options{
		Log:     logger.With("transport", transport.Name()),
		Config:  config.Transport,
		Handler: middleware.Handler(mux),
	})
	if transportErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new transport failed").WithCause(transportErr)))
		return
	}
	// transport <<<

	// proxy >>>
	var proxy proxies.Proxy
	if proxyOptions := opt.proxyOptions; len(proxyOptions) > 0 {
		cluster, ok := manager.(clusters.ClusterEndpointsManager)
		if !ok {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new proxy failed").WithCause(fmt.Errorf("application was not in cluster mode"))))
			return
		}
		var proxyErr error
		proxy, proxyErr = proxies.New(proxyOptions...)
		if proxyErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, new proxy failed").WithCause(proxyErr)))
			return
		}
		constructErr := proxy.Construct(proxies.ProxyOptions{
			Log:     logger.With("fns", "proxy"),
			Config:  config.Proxy,
			Runtime: rt,
			Manager: cluster,
			Dialer:  transport,
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
			Log:    logger.With("hook", hook.Name()),
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
		log:             logger,
		config:          config,
		amp:             amp,
		worker:          worker,
		manager:         manager,
		middlewares:     middleware,
		transport:       transport,
		proxy:           proxy,
		hooks:           opt.hooks,
		shutdownTimeout: opt.shutdownTimeout,
		synced:          false,
		signalCh:        signalCh,
	}
	// deploy
	app.Deploy(builtins...)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type application struct {
	id              string
	name            string
	version         versions.Version
	rt              *runtime.Runtime
	status          *switchs.Switch
	log             logs.Logger
	config          configs.Config
	amp             *procs.AutoMaxProcs
	worker          workers.Workers
	manager         services.EndpointsManager
	middlewares     transports.Middlewares
	transport       transports.Transport
	proxy           proxies.Proxy
	hooks           []hooks.Hook
	shutdownTimeout time.Duration
	synced          bool
	cancel          context.CancelFunc
	signalCh        chan os.Signal
}

func (app *application) Deploy(s ...services.Service) Application {
	for _, service := range s {
		err := app.manager.Add(service)
		if err != nil {
			panic(fmt.Sprintf("%+v", errors.Warning("fns: deploy failed").WithCause(err)))
			return app
		}
	}
	return app
}

func (app *application) Run(ctx context.Context) Application {
	app.amp.Enable()
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
		app.amp.Reset()
		panic(fmt.Sprintf("%+v", errors.Warning("fns: application run failed").WithCause(trErr)))
		return app
	case <-time.After(3 * time.Second):
		break
	}
	if app.log.DebugEnabled() {
		app.log.Debug().With("port", strconv.Itoa(app.transport.Port())).Message("fns: transport is serving...")
	}
	app.status.On()
	app.status.Confirm()

	// endpoints
	lnErr := app.manager.Listen(ctx)
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
		if app.log.DebugEnabled() {
			app.log.Debug().With("port", strconv.Itoa(app.proxy.Port())).Message("fns: proxy is serving...")
		}
	}
	// hooks
	ctx, app.cancel = context.WithCancel(ctx)
	runtime.With(ctx, app.rt)
	for _, hook := range app.hooks {
		name := hook.Name()
		if name == "" {
			if app.log.DebugEnabled() {
				app.log.Debug().Message("fns: one hook has no name")
			}
			continue
		}
		hookConfig, hookConfigErr := app.config.Hooks.Get(name)
		if hookConfigErr != nil {
			if app.log.DebugEnabled() {
				app.log.Debug().With("hook", name).Cause(hookConfigErr).Message("fns: get hook config failed")
			}
			continue
		}
		hookErr := hook.Construct(hooks.Options{
			Log:    app.log.With("hook", name),
			Config: hookConfig,
		})
		if hookErr != nil {
			if app.log.DebugEnabled() {
				app.log.Debug().With("hook", name).Cause(hookErr).Message("fns: construct hook failed")
			}
			continue
		}
		go hook.Execute(ctx)
		if app.log.DebugEnabled() {
			app.log.Debug().With("hook", hook.Name()).Message("fns: hook is dispatched")
		}
	}
	// log
	if app.log.DebugEnabled() {
		app.log.Debug().Message("fns: application is running...")
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
	defer app.amp.Reset()
	timeout := app.shutdownTimeout
	if timeout < 1 {
		timeout = 10 * time.Minute
	}
	app.cancel()
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	// status
	app.status.Off()

	go func(ctx context.Context, cancel context.CancelFunc, app *application) {
		// endpoints
		app.manager.Shutdown(ctx)
		// transport
		app.middlewares.Close()
		app.transport.Shutdown(ctx)
		// proxy
		if app.proxy != nil {
			app.proxy.Shutdown(ctx)
		}
		// log
		app.log.Shutdown(ctx)
		cancel()
	}(context.Wrap(ctx), cancel, app)
	<-ctx.Done()
	app.status.Confirm()
	// log
	if app.log.DebugEnabled() {
		app.log.Debug().Message("fns: application is stopped!!!")
	}
}

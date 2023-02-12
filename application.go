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
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/internal/configure"
	"github.com/aacfactory/fns/internal/logger"
	"github.com/aacfactory/fns/internal/procs"
	"github.com/aacfactory/fns/server"
	"github.com/aacfactory/fns/service"
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
	Run(ctx context.Context) (err error)
	RunWithHooks(ctx context.Context, hook ...Hook) (err error)
	Execute(ctx context.Context, serviceName string, fn string, argument interface{}, options ...ExecuteOption) (result interface{}, err errors.CodeError)
	Sync() (err error)
	Quit()
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
	appId := uid.UID()
	appName := opt.name
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
	// log
	logOptions := logger.LogOptions{
		Name: appName,
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

	// procs
	goprocs := procs.New(procs.Options{
		Log: log,
		Min: opt.autoMaxProcsMin,
		Max: opt.autoMaxProcsMax,
	})

	// endpoints
	endpoints, endpointsErr := service.NewEndpoints(service.EndpointsOptions{
		AppId:  appId,
		Log:    log,
		Config: config.Runtime,
	})
	if endpointsErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(errors.Map(endpointsErr))))
		return
	}

	// http >>>
	httpConfig := config.Server
	if httpConfig == nil {
		httpConfig = configure.DefaultServer()
	}
	// http handlers
	httpHandlersConfigRaw := httpConfig.Handlers
	if httpHandlersConfigRaw == nil || len(httpHandlersConfigRaw) == 0 {
		httpHandlersConfigRaw = []byte{'{', '}'}
	}
	httpHandlersConfig, httpHandlersConfigErr := configures.NewJsonConfig(httpHandlersConfigRaw)
	if httpHandlersConfigErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(errors.Map(httpHandlersConfigErr))))
		return
	}
	httpHandlers, httpHandlersErr := server.NewHandlers(server.HandlersOptions{
		AppId:      appId,
		AppName:    appName,
		AppVersion: appVersion,
		Log:        log,
		Config:     httpHandlersConfig,
		Endpoints:  endpoints,
	})
	if httpHandlersErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(errors.Map(httpHandlersErr))))
		return
	}
	if len(opt.serverHandlers) > 0 {
		for _, handler := range opt.serverHandlers {
			if handler == nil {
				continue
			}
			appendErr := httpHandlers.Append(handler)
			if appendErr != nil {
				panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(errors.Map(appendErr))))
				return
			}
		}
	}
	// http cors
	corsHandler := server.NewCorsHandler(httpConfig.Cors)
	// http handler
	httpHandler := corsHandler.Handler(httpHandlers)
	// http server
	httpServer := opt.server
	if httpServer == nil {
		httpServer = &server.FastHttp{}
	}
	httpOptions, httpOptionsErr := server.NewHttpOptions(httpConfig, log, httpHandler)
	if httpOptionsErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(errors.Map(httpOptionsErr))))
		return
	}
	serverBuildErr := httpServer.Build(httpOptions)
	if serverBuildErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create http server failed").WithCause(serverBuildErr)))
		return
	}
	// http <<<

	// cluster
	var cluster clusters.Cluster
	if config.Cluster != nil {
		clusterName := strings.TrimSpace(config.Cluster.Name)
		clusterOptions := config.Cluster.Options
		if clusterOptions == nil || len(clusterOptions) == 0 {
			clusterOptions = []byte{'{', '}'}
		}
		clusterConfig, clusterConfigErr := configures.NewJsonConfig(config.Cluster.Options)
		if clusterConfigErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create cluster failed").WithCause(fmt.Errorf("config of %s is invalid, %v", clusterName, clusterConfigErr))))
			return
		}
		clusterBuilder, hasClusterBuilder := clusters.GetClusterBuilder(clusterName)
		if !hasClusterBuilder {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create cluster failed").WithCause(fmt.Errorf("%s is not defined", clusterName))))
			return
		}
		var clusterBuildErr error
		cluster, clusterBuildErr = clusterBuilder(clusters.ClusterBuilderOptions{
			Config:    clusterConfig,
			Endpoints: endpoints,
		})
		if clusterBuildErr != nil {
			panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, create cluster failed").WithCause(errors.Map(clusterBuildErr))))
			return
		}
		discovery := cluster.EndpointDiscovery()
		barrier := cluster.Shared().Barrier()
		store := cluster.Shared().Store()
		lockers := cluster.Shared().Lockers()
		endpoints.Upgrade(barrier, store, lockers, discovery)
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
		autoMaxProcs:    goprocs,
		config:          configRaw,
		cluster:         cluster,
		endpoints:       endpoints,
		http:            httpServer,
		httpHandlers:    httpHandlers,
		shutdownTimeout: opt.shutdownTimeout,
		signalCh:        signalCh,
		synced:          false,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type application struct {
	log             logs.Logger
	autoMaxProcs    *procs.AutoMaxProcs
	config          configures.Config
	cluster         clusters.Cluster
	endpoints       *service.Endpoints
	http            server.Http
	httpHandlers    *server.Handlers
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
	}
	return
}

func (app *application) Run(ctx context.Context) (err error) {
	if app.endpoints.IsRunning() {
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
	// listenable services
	serviceListenErr := app.endpoints.Listen()
	if serviceListenErr != nil {
		err = errors.Warning("fns: run application failed").WithCause(serviceListenErr)
		return
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
	case <-time.After(3 * time.Second):
		break
	case httpErr := <-httpListenCh:
		err = httpErr
		return
	}
	// cluster publish
	if app.cluster != nil {
		joinErr := app.cluster.Join(ctx)
		if joinErr != nil {
			err = errors.Warning("fns: run application failed").WithCause(joinErr)
			return
		}
	}
	// on
	app.endpoints.Run()
	return
}

func (app *application) RunWithHooks(ctx context.Context, hooks ...Hook) (err error) {
	runErr := app.Run(ctx)
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
		service.Fork(ctx, hook)
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
	if !app.endpoints.IsRunning() {
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
	// cluster leave
	if app.cluster != nil {
		leaveErr := app.cluster.Leave(context.TODO())
		if leaveErr != nil {
			if app.log.WarnEnabled() {
				app.log.Warn().Cause(leaveErr).Message("fns: an error occurred in the stop application")
			}
		}
	}
	// endpoints
	app.endpoints.Close()
	// http
	app.httpHandlers.Close()
	httpCloseErr := app.http.Close()
	if httpCloseErr != nil {
		if app.log.WarnEnabled() {
			app.log.Warn().Cause(httpCloseErr).Message("fns: an error occurred in the stop application")
		}
	}
	// return
	ch <- struct{}{}
	close(ch)
	return
}

func (app *application) Quit() {
	if !app.endpoints.IsRunning() {
		return
	}
	if !app.synced {
		go func(app *application) {
			_ = app.Sync()
		}(app)
		time.Sleep(1 * time.Second)
	}
	app.signalCh <- syscall.SIGQUIT
}

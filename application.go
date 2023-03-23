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
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/logs"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Application interface {
	Deploy(service ...service.Service) (err error)
	Run(ctx context.Context) (err error)
	RunWithHooks(ctx context.Context, hook ...Hook) (err error)
	Execute(ctx context.Context, serviceName string, fn string, argument interface{}, options ...ExecuteOption) (result service.FutureResult, err errors.CodeError)
	Log() (log logs.Logger)
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
	appId := strings.TrimSpace(opt.id)
	if appId == "" {
		appId = uid.UID()
	}
	appName := opt.name
	appVersion := opt.version
	// config
	configRetriever, configRetrieverErr := configures.NewRetriever(opt.configRetrieverOption)
	if configRetrieverErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed for invalid config retriever").WithCause(configRetrieverErr)))
		return
	}
	config, configGetErr := configRetriever.Get()
	if configGetErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, get config via retriever failed").WithCause(configGetErr)))
		return
	}

	// endpoints
	endpoints, endpointsErr := service.NewEndpoints(service.EndpointsOptions{
		OpenApiVersion: opt.openApiVersion,
		AppId:          appId,
		AppName:        appName,
		AppVersion:     appVersion,
		ProxyMode:      opt.proxyMode,
		Http:           opt.httpEngine,
		HttpHandlers:   opt.httpHandlers,
		Config:         config,
	})
	if endpointsErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed").WithCause(errors.Map(endpointsErr))))
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
		id:              appId,
		name:            appName,
		version:         appVersion,
		config:          config,
		endpoints:       endpoints,
		shutdownTimeout: opt.shutdownTimeout,
		signalCh:        signalCh,
		synced:          false,
	}

	if opt.services != nil && len(opt.services) > 0 {
		for _, svc := range opt.services {
			deployErr := app.Deploy(svc)
			if deployErr != nil {
				panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, deploy service failed").WithCause(errors.Map(deployErr))))
				return
			}
		}
	}
	if opt.httpHandlers != nil && len(opt.httpHandlers) > 0 {
		for _, handler := range opt.httpHandlers {
			handlerWithServices, isHandlerWithServices := handler.(service.HttpHandlerWithServices)
			if isHandlerWithServices {
				handlerServices := handlerWithServices.Services()
				if handlerServices != nil && len(handlerServices) > 0 {
					for _, handlerService := range handlerServices {
						deployErr := app.Deploy(handlerService)
						if deployErr != nil {
							panic(fmt.Errorf("%+v", errors.Warning("fns: new application failed, deploy service failed").WithCause(errors.Map(deployErr))))
							return
						}
					}
				}
			}
		}
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type application struct {
	id              string
	name            string
	version         versions.Version
	config          configures.Config
	endpoints       *service.Endpoints
	shutdownTimeout time.Duration
	signalCh        chan os.Signal
	synced          bool
}

func (app *application) Log() (log logs.Logger) {
	log = app.endpoints.Log()
	return
}

func (app *application) Deploy(services ...service.Service) (err error) {
	if app.endpoints.Running() {
		err = errors.Warning("fns: can not deployed service when application is running")
		return
	}
	if services == nil || len(services) == 0 {
		err = errors.Warning("fns: no services deployed")
		return
	}
	for _, svc := range services {
		if svc == nil {
			err = errors.Warning("fns: deploy service failed for one of services is nil")
			return
		}
		err = app.endpoints.Deploy(svc)
		if err != nil {
			return
		}
	}
	return
}

func (app *application) Run(ctx context.Context) (err error) {
	if app.endpoints.Running() {
		err = errors.Warning("fns: application is running")
		return
	}
	err = app.endpoints.Listen(ctx)
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
	ctx = service.Todo(ctx, app.endpoints)

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
			Log:    app.Log().With("hook", hook.Name()),
			Config: hookConfig,
		})
		if buildErr != nil {
			err = errors.Warning("fns run with hooks failed").WithCause(buildErr)
			return
		}

		service.Fork(ctx, hook)
	}
	return
}

func (app *application) Execute(ctx context.Context, serviceName string, fn string, argument interface{}, options ...ExecuteOption) (result service.FutureResult, err errors.CodeError) {
	if serviceName == "" || fn == "" {
		err = errors.Warning("fns: execute failed").WithCause(fmt.Errorf("service name or fn is invalid"))
		return
	}
	ctx = service.Todo(ctx, app.endpoints)
	endpoint, hasEndpoint := app.endpoints.Get(ctx, serviceName)
	if !hasEndpoint {
		err = errors.Warning("fns: execute failed").WithCause(fmt.Errorf("service was not found")).WithMeta("service", serviceName)
		return
	}

	opt := &ExecuteOptions{}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	requestOptions := make([]service.RequestOption, 0, 1)
	if opt.user != nil {
		requestOptions = append(requestOptions, service.WithRequestUser(opt.user.Id(), opt.user.Attributes()))
	}
	if opt.internal {
		requestOptions = append(requestOptions, service.WithInternalRequest())
	}
	requestOptions = append(requestOptions, service.WithDeviceId(app.id))
	result, err = endpoint.RequestSync(ctx, service.NewRequest(ctx, serviceName, fn, service.NewArgument(argument), requestOptions...))
	if err != nil {
		err = errors.Warning("fns: execute failed").WithCause(err).WithMeta("service", serviceName)
	}
	return
}

func (app *application) Sync() (err error) {
	if app.synced {
		return
	}
	if !app.endpoints.Running() {
		err = errors.Warning("fns: application is not running")
		return
	}
	app.synced = true
	<-app.signalCh
	stopped := make(chan struct{}, 1)
	ctx, cancel := context.WithTimeout(context.TODO(), app.shutdownTimeout)
	go app.stop(ctx, stopped)
	select {
	case <-time.After(app.shutdownTimeout):
		err = errors.Warning("fns: stop application timeout")
		break
	case <-stopped:
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message("fns: stop application succeed")
		}
		break
	}
	cancel()
	return
}

func (app *application) stop(ctx context.Context, ch chan struct{}) {
	// endpoints
	app.endpoints.Close(ctx)
	// return
	ch <- struct{}{}
	close(ch)
	return
}

func (app *application) Quit() {
	if !app.endpoints.Running() {
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

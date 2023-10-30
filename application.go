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
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/logs"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Application interface {
	Deploy(service ...services.Service) (err error)
	Run(ctx context.Context) (err error)
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

	// proxy
	var proxyOptions *services.TransportOptions
	if opt.proxyOptions != nil {
		proxyOptions = &services.TransportOptions{
			Middlewares: opt.proxyOptions.middlewares,
			Handlers:    opt.proxyOptions.handlers,
		}
	}
	// endpoints
	endpoints, endpointsErr := services.NewEndpoints(services.EndpointsOptions{
		AppId:      appId,
		AppName:    appName,
		AppVersion: appVersion,
		Transport: &services.TransportOptions{
			Transport:   opt.transportOptions.transport,
			Middlewares: opt.transportOptions.middlewares,
			Handlers:    opt.transportOptions.handlers,
		},
		Proxy:  proxyOptions,
		Config: config,
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
		hooks:           opt.hooks,
		shutdownTimeout: opt.shutdownTimeout,
		signalCh:        signalCh,
		synced:          false,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type application struct {
	id              string
	name            string
	version         versions.Version
	config          configures.Config
	endpoints       *services.Endpoints
	hooks           []Hook
	shutdownTimeout time.Duration
	signalCh        chan os.Signal
	synced          bool
}

func (app *application) Log() (log logs.Logger) {
	log = app.endpoints.Log()
	return
}

func (app *application) Deploy(services ...services.Service) (err error) {
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
	ctx = app.wrapCtx(ctx)
	err = app.endpoints.Listen(ctx)
	if err != nil {
		err = errors.Warning("fns: run application failed").WithCause(err)
		return
	}
	if app.hooks != nil && len(app.hooks) > 0 {
		config, hasConfig := app.config.Node("hooks")
		if !hasConfig {
			config, _ = configures.NewJsonConfig([]byte{'{', '}'})
		}
		failed := false
		for _, hook := range app.hooks {
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
				failed = true
				err = errors.Warning("fns: run application failed").WithCause(buildErr)
				break
			}
			services.Fork(ctx, hook)
		}
		if failed {
			_ = app.stop(ctx)
			return
		}
	}
	if app.Log().DebugEnabled() {
		app.Log().Debug().Message("fns: run application succeed")
	}
	return
}

func (app *application) Execute(ctx context.Context, serviceName string, fn string, argument interface{}, options ...ExecuteOption) (result services.FutureResult, err errors.CodeError) {
	if serviceName == "" || fn == "" {
		err = errors.Warning("fns: application execute service's fn failed").WithCause(fmt.Errorf("service name or fn is invalid"))
		return
	}
	ctx = app.wrapCtx(ctx)
	endpoint, hasEndpoint := app.endpoints.Get(ctx, serviceName)
	if !hasEndpoint {
		err = errors.Warning("fns: application execute service's fn failed").WithCause(fmt.Errorf("service was not found")).WithMeta("service", serviceName)
		return
	}

	opt := &ExecuteOptions{}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	requestOptions := make([]services.RequestOption, 0, 1)
	if opt.user != nil {
		requestOptions = append(requestOptions, services.WithRequestUser(opt.user.Id(), opt.user.Attributes()))
	}
	if opt.internal {
		requestOptions = append(requestOptions, services.WithInternalRequest())
	}
	requestOptions = append(requestOptions, services.WithDeviceId(app.id))
	result, err = endpoint.RequestSync(ctx, services.NewRequest(ctx, serviceName, fn, services.NewArgument(argument), requestOptions...))
	if err != nil {
		err = errors.Warning("fns: application execute failed").WithCause(err).WithMeta("service", serviceName)
	}
	return
}

func (app *application) wrapCtx(ctx context.Context) context.Context {
	return app.endpoints.Runtime().SetIntoContext(ctx)
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
	err = app.stop(context.TODO())
	return
}

func (app *application) stop(ctx context.Context) (err error) {
	ch := make(chan struct{}, 1)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.TODO(), app.shutdownTimeout)
	go func(ctx context.Context, app *application, ch chan struct{}) {
		// endpoints
		app.endpoints.Close(ctx)
		// return
		ch <- struct{}{}
		close(ch)
	}(ctx, app, ch)
	select {
	case <-time.After(app.shutdownTimeout):
		err = errors.Warning("fns: stop application timeout")
		break
	case <-ch:
		if app.Log().DebugEnabled() {
			app.Log().Debug().Message("fns: stop application succeed")
		}
		break
	}
	cancel()
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

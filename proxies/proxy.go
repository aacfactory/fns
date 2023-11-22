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

package proxies

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/fast"
	"github.com/aacfactory/logs"
)

type ProxyOptions struct {
	Log     logs.Logger
	Config  Config
	Runtime *runtime.Runtime
	Manager clusters.ClusterEndpointsManager
	Dialer  transports.Dialer
}

type Proxy interface {
	Construct(options ProxyOptions) (err error)
	Port() int
	Run(ctx context.Context) (err error)
	Shutdown(ctx context.Context)
}

func New(options ...Option) (p Proxy, err error) {
	opt := Options{
		transport:   fast.New(),
		middlewares: make([]transports.Middleware, 0, 1),
		handlers:    make([]transports.MuxHandler, 0, 1),
	}
	for _, option := range options {
		optErr := option(&opt)
		if optErr != nil {
			err = errors.Warning("fns: new proxy failed").WithCause(optErr)
			return
		}
	}
	p = &proxy{
		log:         nil,
		transport:   opt.transport,
		middlewares: opt.middlewares,
		handlers:    opt.handlers,
	}
	return
}

type proxy struct {
	log         logs.Logger
	transport   transports.Transport
	middlewares []transports.Middleware
	handlers    []transports.MuxHandler
}

func (p *proxy) Construct(options ProxyOptions) (err error) {
	p.log = options.Log
	// config
	config := options.Config
	// builtins
	builtins := make([]services.Service, 0, 1)
	// middlewares
	middlewares := make([]transports.Middleware, 0, 1)
	middlewares = append(middlewares, runtime.Middleware(options.Runtime))
	var corsMiddleware transports.Middleware
	for _, middleware := range p.middlewares {
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
	// handlers
	mux := transports.NewMux()
	for _, handler := range p.handlers {
		handlerConfig, handlerConfigErr := config.HandlerConfig(handler.Name())
		if handlerConfigErr != nil {
			err = errors.Warning("fns: construct proxy failed, new transport handler failed").WithCause(handlerConfigErr).WithMeta("handler", handler.Name())
			return
		}
		handlerErr := handler.Construct(transports.MuxHandlerOptions{
			Log:    p.log.With("handler", handler.Name()),
			Config: handlerConfig,
		})
		if handlerErr != nil {
			err = errors.Warning("fns: construct proxy failed, new transport handler failed").WithCause(handlerErr).WithMeta("handler", handler.Name())
			return
		}
		mux.Add(handler)
		builtin, isBuiltin := handler.(services.MuxHandler)
		if isBuiltin {
			builtins = append(builtins, builtin.Services()...)
		}
	}
	mux.Add(NewProxyHandler(options.Manager, options.Dialer))
	// transport
	transportErr := p.transport.Construct(transports.Options{
		Log:         p.log.With("transport", p.transport.Name()),
		Config:      config.Config,
		Middlewares: middlewares,
		Handler:     mux,
	})
	if transportErr != nil {
		err = errors.Warning("fns: construct proxy failed, new transport failed").WithCause(transportErr)
		return
	}
	for _, builtin := range builtins {
		err = options.Manager.Add(builtin)
		if err != nil {
			err = errors.Warning("fns: construct proxy failed, deploy service failed").WithCause(err)
			return
		}
	}
	return
}

func (p *proxy) Port() int {
	return p.transport.Port()
}

func (p *proxy) Run(_ context.Context) (err error) {
	err = p.transport.ListenAndServe()
	if err != nil {
		err = errors.Warning("fns: proxy run failed").WithCause(err)
		return
	}
	return
}

func (p *proxy) Shutdown(ctx context.Context) {
	p.transport.Shutdown(ctx)
	return
}

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

package server

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"sync"
)

const (
	httpConnectionHeader      = "Connection"
	httpConnectionHeaderClose = "close"
)

type HandlerOptions struct {
	AppId             string
	AppName           string
	AppVersion        versions.Version
	Log               logs.Logger
	Config            configures.Config
	DeployedEndpoints service.DeployedEndpoints
}

type Handler interface {
	http.Handler
	Name() (name string)
	Build(options *HandlerOptions) (err error)
	Accept(request *http.Request) (ok bool)
	Close()
}

type HandlersOptions struct {
	AppId             string
	AppName           string
	AppVersion        versions.Version
	Log               logs.Logger
	Config            configures.Config
	DeployedEndpoints service.DeployedEndpoints
	Running           *commons.SafeFlag
}

func NewHandlers(options HandlersOptions) (handlers *Handlers, err errors.CodeError) {
	handlers = &Handlers{
		appId:             options.AppId,
		appName:           options.AppName,
		appVersion:        options.AppVersion,
		log:               options.Log,
		config:            options.Config,
		deployedEndpoints: options.DeployedEndpoints,
		handlers:          make([]Handler, 0, 1),
		running:           options.Running,
	}
	// health
	err = handlers.Append(&healthHandler{})
	if err != nil {
		err = errors.Warning("fns: create handlers failed").WithCause(errors.Map(err))
		return
	}
	// documents
	err = handlers.Append(&documentHandler{})
	if err != nil {
		err = errors.Warning("fns: create handlers failed").WithCause(errors.Map(err))
		return
	}
	// service
	err = handlers.Append(&serviceHandler{})
	if err != nil {
		err = errors.Warning("fns: create handlers failed").WithCause(errors.Map(err))
		return
	}
	return
}

type Handlers struct {
	appId             string
	appName           string
	appVersion        versions.Version
	log               logs.Logger
	config            configures.Config
	deployedEndpoints service.DeployedEndpoints
	running           *commons.SafeFlag
	handlers          []Handler
}

func (handlers *Handlers) Append(h Handler) (err errors.CodeError) {
	if h == nil {
		return
	}
	name := h.Name()
	handleConfig, hasHandleConfig := handlers.config.Node(name)
	if !hasHandleConfig {
		handleConfig, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	options := &HandlerOptions{
		AppId:             handlers.appId,
		AppName:           handlers.appName,
		AppVersion:        handlers.appVersion,
		Log:               handlers.log.With("handler", name),
		Config:            handleConfig,
		DeployedEndpoints: handlers.deployedEndpoints,
	}
	buildErr := h.Build(options)
	if buildErr != nil {
		err = errors.Warning("fns: build handler failed").WithMeta("name", name).WithCause(errors.Map(buildErr))
		return
	}
	handlers.handlers = append(handlers.handlers, h)
	return
}

func (handlers *Handlers) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if handlers.running.IsOff() {
		writer.Header().Set(httpConnectionHeader, httpConnectionHeaderClose)
		writer.Header().Set(httpServerHeader, httpServerHeaderValue)
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write(json.UnsafeMarshal(errors.Unavailable("fns: service is unavailable").WithMeta("fns", "handlers")))
		return
	}
	handled := false
	for _, handler := range handlers.handlers {
		if handler.Accept(request) {
			handler.ServeHTTP(writer, request)
			handled = true
			break
		}
	}
	if !handled {
		writer.Header().Set(httpServerHeader, httpServerHeaderValue)
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusNotAcceptable)
		_, _ = writer.Write(json.UnsafeMarshal(errors.NotAcceptable("fns: request is not accepted").WithMeta("fns", "handlers")))
		return
	}
	return
}

func (handlers *Handlers) HandlerNames() (names []string) {
	names = make([]string, 0, 1)
	for _, handler := range handlers.handlers {
		names = append(names, handler.Name())
	}
	return
}

func (handlers *Handlers) Close() {
	waiter := &sync.WaitGroup{}
	for _, handler := range handlers.handlers {
		waiter.Add(1)
		go func(handler Handler, waiter *sync.WaitGroup) {
			handler.Close()
			waiter.Done()
		}(handler, waiter)
	}
	waiter.Wait()
}

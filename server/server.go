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
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/fns/internal/logger"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	httpServerHeader      = "Server"
	httpServerHeaderValue = "FNS"
	httpContentType       = "Content-Type"
	httpContentTypeJson   = "application/json"
)

type Handler interface {
	Handle(writer http.ResponseWriter, request *http.Request) (ok bool)
	Close()
}

type InterceptorHandlerOptions struct {
	Log    logs.Logger
	Config configuares.Config
}

type InterceptorHandler interface {
	Handler
	Build(options InterceptorHandlerOptions) (err error)
	Name() (name string)
}

func NewHandlers() (handlers *Handlers) {
	handlers = &Handlers{
		handlers: make([]Handler, 0, 1),
	}
	return
}

type Handlers struct {
	handlers []Handler
}

func (handlers *Handlers) Append(h Handler) {
	if h == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: append handler into handler chain failed cause handler is nil")))
	}
	handlers.handlers = append(handlers.handlers, h)
}

func (handlers *Handlers) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handled := false
	for _, handler := range handlers.handlers {
		if handler.Handle(writer, request) {
			handled = true
			break
		}
	}
	if !handled {
		writer.Header().Set(httpServerHeader, httpServerHeaderValue)
		writer.Header().Set(httpContentType, httpContentTypeJson)
		writer.WriteHeader(http.StatusNotImplemented)
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

type HttpClient interface {
	Do(ctx context.Context, method string, url string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error)
	Close()
}

type HttpOptions struct {
	Port      int
	ServerTLS *tls.Config
	ClientTLS *tls.Config
	Handler   http.Handler
	Log       logs.Logger
	Raw       *json.Object
}

func (options HttpOptions) GetOption(key string, value interface{}) (has bool, err error) {
	has = options.Raw.Contains(key)
	if !has {
		return
	}
	err = options.Raw.Get(key, value)
	if err != nil {
		err = errors.Warning(fmt.Sprintf("fns: http server options get %s failed", key)).WithCause(err).WithMeta("fns", "http")
		return
	}
	return
}

type Http interface {
	Build(options HttpOptions) (err error)
	ListenAndServe() (err error)
	Close() (err error)
}

type FastHttp struct {
	log logs.Logger
	ln  net.Listener
	srv *fasthttp.Server
}

func (srv *FastHttp) Build(options HttpOptions) (err error) {
	srv.log = options.Log.With("fns", "http")

	var ln net.Listener
	if options.ServerTLS == nil {
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", options.Port))
	} else {
		ln, err = tls.Listen("tcp", fmt.Sprintf(":%d", options.Port), options.ServerTLS)
	}
	if err != nil {
		err = errors.Warning("fns: build server failed").WithCause(err).WithMeta("fns", "http")
		return
	}
	srv.ln = ln

	var optionErr error
	readTimeoutSeconds := 0
	_, optionErr = options.GetOption("readTimeoutSeconds", &readTimeoutSeconds)
	if optionErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optionErr).WithMeta("fns", "http")
		return
	}
	if readTimeoutSeconds < 1 {
		readTimeoutSeconds = 2
	}
	maxWorkerIdleSeconds := 0
	_, optionErr = options.GetOption("maxWorkerIdleSeconds", &maxWorkerIdleSeconds)
	if optionErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optionErr).WithMeta("fns", "http")
		return
	}
	if maxWorkerIdleSeconds < 1 {
		maxWorkerIdleSeconds = 10
	}
	maxRequestBody := ""
	_, optionErr = options.GetOption("maxRequestBody", &maxRequestBody)
	if optionErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optionErr).WithMeta("fns", "http")
		return
	}
	maxRequestBody = strings.ToUpper(strings.TrimSpace(maxRequestBody))
	if maxRequestBody == "" {
		maxRequestBody = "4MB"
	}
	maxRequestBodySize, maxRequestBodySizeErr := commons.ToBytes(maxRequestBody)
	if maxRequestBodySizeErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(maxRequestBodySizeErr).WithMeta("fns", "http")
		return
	}
	reduceMemoryUsage := false
	_, optionErr = options.GetOption("reduceMemoryUsage", &reduceMemoryUsage)
	if optionErr != nil {
		err = errors.Warning("fns: build server failed").WithCause(optionErr).WithMeta("fns", "http")
		return
	}
	srv.srv = &fasthttp.Server{
		Handler:                            fasthttpadaptor.NewFastHTTPHandler(options.Handler),
		ErrorHandler:                       fastHttpErrorHandler,
		ReadTimeout:                        time.Duration(readTimeoutSeconds) * time.Second,
		MaxIdleWorkerDuration:              time.Duration(maxWorkerIdleSeconds) * time.Second,
		MaxRequestBodySize:                 int(maxRequestBodySize),
		ReduceMemoryUsage:                  reduceMemoryUsage,
		DisablePreParseMultipartForm:       true,
		SleepWhenConcurrencyLimitsExceeded: 10 * time.Second,
		NoDefaultServerHeader:              true,
		NoDefaultDate:                      false,
		NoDefaultContentType:               false,
		CloseOnShutdown:                    true,
		Logger:                             &logger.Printf{Core: options.Log},
	}
	return
}

func (srv *FastHttp) ListenAndServe() (err error) {
	err = srv.srv.Serve(srv.ln)
	if err != nil {
		err = errors.Warning("fns: server listen and serve failed").WithCause(err).WithMeta("fns", "http")
		return
	}
	return
}

func (srv *FastHttp) Close() (err error) {
	err = srv.srv.Shutdown()
	if err != nil {
		err = errors.Warning("fns: server close failed").WithCause(err).WithMeta("fns", "http")
	}
	return
}

func fastHttpErrorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(555)
	ctx.SetContentType(httpContentTypeJson)
	ctx.SetBody([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
}

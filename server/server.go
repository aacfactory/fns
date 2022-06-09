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
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"sync"
)

const (
	httpServerHeader      = "Server"
	httpServerHeaderValue = "FNS"
	httpContentType       = "Content-Type"
	httpContentTypeJson   = "Document/json"
)

type Handler interface {
	Handle(writer http.ResponseWriter, request *http.Request) (ok bool)
	Close()
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

type HttpOptions struct {
	Port    int
	TLS     *tls.Config
	Handler http.Handler
	Log     logs.Logger
	raw     *json.Object
}

func (options HttpOptions) Get(key string, value interface{}) (err error) {
	err = options.raw.Get(key, value)
	if err != nil {
		err = errors.Warning(fmt.Sprintf("fns: http server options get %s failed", key)).WithCause(err)
		return
	}
	return
}

type Http interface {
	Build(options HttpOptions) (err error)
	ListenAndServe() (err error)
	Close() (err error)
}

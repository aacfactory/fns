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

package transports

import (
	"context"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/service/ssl"
	"github.com/aacfactory/logs"
	"io"
)

type Options struct {
	Port    int
	TLS     ssl.Config
	Handler Handler
	Log     logs.Logger
	Config  configures.Config
}

type Client interface {
	Do(ctx context.Context, request *Request) (response *Response, err error)
}

type Dialer interface {
	Dial(address string) (client Client, err error)
}

type Transport interface {
	Name() (name string)
	Build(options Options) (err error)
	Dialer
	ListenAndServe() (err error)
	io.Closer
}

type HandlerBuilder func() Handler

type Handler interface {
	Handle(w ResponseWriter, r *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) Handle(w ResponseWriter, r *Request) {
	f(w, r)
}

type MiddlewareOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Middleware interface {
	Name() string
	Build(options MiddlewareOptions) (err error)
	Handler(next Handler) Handler
}

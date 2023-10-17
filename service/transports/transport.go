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
	"github.com/aacfactory/logs"
)

type Options struct {
	Log                logs.Logger
	Config             *Config
	MiddlewareBuilders []MiddlewareBuilder
	Handler            Handler
}

type Client interface {
	Do(ctx context.Context, request *Request) (response *Response, err error)
}

type Dialer interface {
	Dial(address string) (client Client, err error)
}

type Server interface {
	Port() (port int)
	ListenAndServe() (err error)
	Shutdown() (err error)
}

type Transport interface {
	Name() (name string)
	Build(options Options) (err error)
	Dialer
	Server
}

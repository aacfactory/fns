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
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/service/ssl"
	"github.com/aacfactory/logs"
	"io"
)

var (
	registered = make(map[string]Transport)
)

func Register(transport Transport) {
	registered[transport.Name()] = transport
}

func Registered(name string) (transport Transport, has bool) {
	if name == "" {
		transport = &fastHttpTransport{}
		has = true
		return
	}
	transport, has = registered[name]
	return
}

type Options struct {
	Port    int
	TLS     ssl.Config
	Handler Handler
	Log     logs.Logger
	Config  configures.Config
}

type Transport interface {
	Name() (name string)
	Build(options Options) (err error)
	Dialer
	ListenAndServe() (err error)
	io.Closer
}

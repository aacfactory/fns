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
	"crypto/tls"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/logs"
	"io"
	"net/http"
)

func NewOptions() {

}

type Options struct {
	Port      int
	ServerTLS *tls.Config
	ClientTLS *tls.Config
	Handler   http.Handler
	Log       logs.Logger
	Options   configures.Config
}

type Transport interface {
	Name() (name string)
	Build(options Options) (err error)
	Dialer
	ListenAndServe() (err error)
	io.Closer
}

type Dialer interface {
	Dial(address string) (client Client, err error)
}

type Client interface {
	Get(ctx context.Context, path string, header http.Header) (status int, respHeader http.Header, respBody []byte, err error)
	Post(ctx context.Context, path string, header http.Header, body []byte) (status int, respHeader http.Header, respBody []byte, err error)
	Close()
}

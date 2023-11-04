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
	sc "context"
	"crypto/tls"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
)

var (
	ErrTooBigRequestBody = errors.Warning("fns: request body is too big")
)

type Request interface {
	context.Context
	TLS() bool
	TLSConnectionState() *tls.ConnectionState
	RemoteAddr() []byte
	Proto() []byte
	Host() []byte
	Method() []byte
	Header() Header
	Path() []byte
	Params() Params
	Body() ([]byte, error)
}

func LoadRequest(ctx sc.Context) Request {
	r, ok := ctx.(Request)
	if ok {
		return r
	}
	return nil
}

func TryLoadRequest(ctx sc.Context) (Request, bool) {
	r, ok := ctx.(Request)
	return r, ok
}

func TryLoadRequestHeader(ctx sc.Context) (Header, bool) {
	r, ok := ctx.(Request)
	if !ok {
		return nil, false
	}
	return r.Header(), ok
}

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

package transports

import (
	"crypto/tls"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"net/http"
)

var (
	ErrTooBigRequestBody = errors.New(http.StatusRequestEntityTooLarge, "***TOO LARGE BODY***", "fns: request body is too large")
)

var (
	MethodGet  = []byte(http.MethodGet)
	MethodPost = []byte(http.MethodPost)
)

type Request interface {
	context.Context
	TLS() bool
	TLSConnectionState() *tls.ConnectionState
	RemoteAddr() []byte
	Proto() []byte
	Host() []byte
	Method() []byte
	SetMethod(method []byte)
	Header() Header
	Cookie(key []byte) (value []byte)
	SetCookie(key []byte, value []byte)
	Path() []byte
	Params() Params
	FormValue(name []byte) (value []byte)
	Body() ([]byte, error)
	SetBody(body []byte)
}

const (
	requestContextKey = "@fns:context:transports:request"
)

func WithRequest(ctx context.Context, r Request) context.Context {
	ctx.SetLocalValue(bytex.FromString(requestContextKey), r)
	return ctx
}

func TryLoadRequest(ctx context.Context) (Request, bool) {
	r, ok := ctx.(Request)
	if ok {
		return r, ok
	}
	v := ctx.LocalValue(bytex.FromString(requestContextKey))
	if v == nil {
		return nil, false
	}
	r, ok = v.(Request)
	return r, ok
}

func LoadRequest(ctx context.Context) Request {
	r, ok := TryLoadRequest(ctx)
	if ok {
		return r
	}
	return nil
}

func TryLoadRequestHeader(ctx context.Context) (Header, bool) {
	r, ok := TryLoadRequest(ctx)
	if !ok {
		return nil, false
	}
	return r.Header(), ok
}

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

package fast

import (
	"crypto/tls"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
)

type Request struct {
	*Context
}

func (r *Request) TLS() bool {
	return r.Context.IsTLS()
}

func (r *Request) TLSConnectionState() *tls.ConnectionState {
	return r.Context.TLSConnectionState()
}

func (r *Request) RemoteAddr() []byte {
	return bytex.FromString(r.Context.RemoteAddr().String())
}

func (r *Request) Proto() []byte {
	return r.Context.Request.Header.Protocol()
}

func (r *Request) Host() []byte {
	return r.Context.Host()
}

func (r *Request) Method() []byte {
	return r.Context.Method()
}

func (r *Request) SetMethod(method []byte) {
	r.Context.Request.Header.SetMethodBytes(method)
}

func (r *Request) Cookie(key []byte) (value []byte) {
	value = r.Context.Request.Header.CookieBytes(key)
	return
}

func (r *Request) SetCookie(key []byte, value []byte) {
	r.Context.Request.Header.SetCookieBytesKV(key, value)
}

func (r *Request) Header() transports.Header {
	return RequestHeader{&r.Context.Request.Header}
}

func (r *Request) Path() []byte {
	return r.Context.URI().Path()
}

func (r *Request) Params() transports.Params {
	return &Params{args: r.Context.QueryArgs()}
}

func (r *Request) FormValue(name []byte) (value []byte) {
	return r.Context.FormValue(string(name))
}

func (r *Request) Body() ([]byte, error) {
	return r.Context.PostBody(), nil
}

func (r *Request) SetBody(body []byte) {
	r.Context.Request.SetBody(body)
}

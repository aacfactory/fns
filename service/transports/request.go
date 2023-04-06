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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/valyala/bytebufferpool"
	"net/http"
	"net/url"
	"sort"
)

var (
	ErrTooBigRequestBody = errors.Warning("fns: request body is too big")
)

const (
	httpSchema  = "http"
	httpsSchema = "https"
)

type RequestParams map[string][]byte

func (params RequestParams) Add(name []byte, value []byte) {
	if name == nil || value == nil {
		return
	}
	if len(name) == 0 {
		return
	}
	params[bytex.ToString(name)] = value
}

func (params RequestParams) Get(name []byte) []byte {
	if name == nil {
		return nil
	}
	if len(name) == 0 {
		return nil
	}
	value, has := params[bytex.ToString(name)]
	if !has {
		return nil
	}
	return value
}

func (params RequestParams) Del(name []byte) {
	if name == nil {
		return
	}
	if len(name) == 0 {
		return
	}
	delete(params, bytex.ToString(name))
}

func (params RequestParams) String() string {
	size := len(params)
	if size == 0 {
		return ""
	}
	names := make([]string, 0, size)
	for name := range params {
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	buf := bytebufferpool.Get()
	for _, name := range names {
		_, _ = buf.WriteString(fmt.Sprintf("&%s=%s", name, bytex.ToString(params[name])))
	}
	s := bytex.ToString(buf.Bytes()[1:])
	bytebufferpool.Put(buf)
	return s
}

func NewRequest(ctx context.Context, method []byte, uri []byte) (r *Request, err error) {
	u, parseURIErr := url.ParseRequestURI(bytex.ToString(uri))
	if parseURIErr != nil {
		err = errors.Warning("fns: new transport request failed").WithCause(parseURIErr)
		return
	}
	r = &Request{
		ctx:        ctx,
		isTLS:      false,
		method:     method,
		host:       nil,
		remoteAddr: nil,
		header:     make(Header),
		path:       bytex.FromString(u.Path),
		params:     make(RequestParams),
		body:       nil,
	}

	return
}

type Request struct {
	ctx        context.Context
	isTLS      bool
	method     []byte
	host       []byte
	remoteAddr []byte
	header     Header
	path       []byte
	params     RequestParams
	body       []byte
}

func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

func (r *Request) Context() context.Context {
	return r.ctx
}

func (r *Request) IsTLS() bool {
	return r.isTLS
}

func (r *Request) UseTLS() {
	r.isTLS = true
}

func (r *Request) Method() []byte {
	return r.method
}

func (r *Request) IsGet() bool {
	return bytex.ToString(r.method) == http.MethodGet
}

func (r *Request) IsPost() bool {
	return bytex.ToString(r.method) == http.MethodPost
}

func (r *Request) RemoteAddr() []byte {
	return r.remoteAddr
}

func (r *Request) Host() []byte {
	return r.host
}

func (r *Request) SetHost(host []byte) {
	r.host = host
}

func (r *Request) Header() Header {
	return r.header
}

func (r *Request) Path() []byte {
	return r.path
}

func (r *Request) Param(name string) []byte {
	return r.params[name]
}

func (r *Request) Params() RequestParams {
	return r.params
}

func (r *Request) Body() []byte {
	return r.body
}

func (r *Request) SetBody(body []byte) {
	r.body = body
}

func (r *Request) URL() ([]byte, error) {
	if r.host == nil || len(r.host) == 0 {
		return nil, errors.Warning("fns: make transport request url failed").WithCause(errors.Warning("host is required"))
	}
	if r.path == nil || len(r.path) == 0 {
		return nil, errors.Warning("fns: make transport request url failed").WithCause(errors.Warning("path is required"))
	}
	schema := httpSchema
	if r.isTLS {
		schema = httpsSchema
	}
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	_, _ = buf.Write(bytex.FromString(schema))
	_, _ = buf.Write(bytex.FromString("://"))
	_, _ = buf.Write(r.host)
	_, _ = buf.Write(r.path)
	if r.params != nil && len(r.params) > 0 {
		_, _ = buf.Write([]byte{'?'})
		_, _ = buf.Write(bytex.FromString(r.params.String()))
	}
	return buf.Bytes(), nil
}

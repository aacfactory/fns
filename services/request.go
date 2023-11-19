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

package services

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/json"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"strconv"
	"sync"
)

type Param interface {
	scanner.Scanner
}

func NewParam(src interface{}) Param {
	return scanner.New(src)
}

// +-------------------------------------------------------------------------------------------------------------------+

type RequestOption func(*RequestOptions)

func WithRequestId(id []byte) RequestOption {
	return func(options *RequestOptions) {
		options.header.requestId = id
	}
}

func WithProcessId(id []byte) RequestOption {
	return func(options *RequestOptions) {
		options.header.processId = id
	}
}

func WithEndpointId(id []byte) RequestOption {
	return func(options *RequestOptions) {
		options.header.endpointId = id
	}
}

func WithToken(token []byte) RequestOption {
	return func(options *RequestOptions) {
		options.header.token = token
	}
}

func WithDeviceId(id []byte) RequestOption {
	return func(options *RequestOptions) {
		options.header.deviceId = id
	}
}

func WithDeviceIp(ip []byte) RequestOption {
	return func(options *RequestOptions) {
		options.header.deviceIp = ip
	}
}

func WithInternalRequest() RequestOption {
	return func(options *RequestOptions) {
		options.header.internal = true
	}
}

func WithRequestVersions(acceptedVersions versions.Intervals) RequestOption {
	return func(options *RequestOptions) {
		options.header.acceptedVersions = acceptedVersions
	}
}

type RequestOptions struct {
	header Header
}

// +-------------------------------------------------------------------------------------------------------------------+

var (
	requestPool = sync.Pool{}
)

type Request interface {
	context.Context
	Fn() (endpoint []byte, fn []byte)
	Header() (header Header)
	Param() (param Param)
}

func AcquireRequest(ctx context.Context, service []byte, fn []byte, param interface{}, options ...RequestOption) (v Request) {
	opt := &RequestOptions{
		header: Header{},
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	if len(opt.header.processId) == 0 {
		opt.header.processId = uid.Bytes()
	}
	parent, hasParent := TryLoadRequest(ctx)
	if hasParent {
		header := parent.Header()
		if len(opt.header.requestId) == 0 && len(header.requestId) > 0 {
			opt.header.requestId = header.requestId
		}
		if len(opt.header.deviceId) == 0 && len(header.deviceId) > 0 {
			opt.header.deviceId = header.deviceId
		}
		if len(opt.header.deviceIp) == 0 && len(header.deviceIp) > 0 {
			opt.header.deviceIp = header.deviceIp
		}
		if len(opt.header.token) == 0 && len(header.token) > 0 {
			opt.header.token = header.token
		}
		if len(opt.header.acceptedVersions) == 0 && len(header.acceptedVersions) > 0 {
			opt.header.acceptedVersions = header.acceptedVersions
		}
		opt.header.internal = true
	}
	var r *request
	cached := requestPool.Get()
	if cached == nil {
		r = new(request)
	} else {
		r = cached.(*request)
	}
	r.Context = context.Acquire(ctx)
	r.header = opt.header
	r.service = service
	r.fn = fn
	r.param = NewParam(param)
	v = r
	return
}

func ReleaseRequest(r Request) {
	req, ok := r.(*request)
	if !ok {
		return
	}
	context.Release(req)
	req.Context = nil
	requestPool.Put(req)
}

type request struct {
	context.Context
	header  Header
	service []byte
	fn      []byte
	param   Param
}

func (r *request) Fn() (service []byte, fn []byte) {
	service, fn = r.service, r.fn
	return
}

func (r *request) Header() (header Header) {
	header = r.header
	return
}

func (r *request) Param() (param Param) {
	param = r.param
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func TryLoadRequest(ctx context.Context) (r Request, ok bool) {
	r, ok = ctx.(Request)
	return
}

func LoadRequest(ctx context.Context) Request {
	r, ok := ctx.(Request)
	if ok {
		return r
	}
	panic(fmt.Sprintf("%+v", errors.Warning("fns: can not convert context to request")))
	return r
}

// +-------------------------------------------------------------------------------------------------------------------+

type HashRequestOptions struct {
	withToken bool
}

type HashRequestOption func(options *HashRequestOptions)

func HashRequestWithToken() HashRequestOption {
	return func(options *HashRequestOptions) {
		options.withToken = true
	}
}

func HashRequest(r Request, options ...HashRequestOption) (p []byte, err error) {
	opt := HashRequestOptions{
		withToken: false,
	}
	for _, option := range options {
		option(&opt)
	}
	service, fn := r.Fn()
	pp, encodeErr := json.Marshal(r.Param())
	if encodeErr != nil {
		err = errors.Warning("fns: hash request failed").WithCause(encodeErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}
	buf := bytebufferpool.Get()
	_, _ = buf.Write(service)
	_, _ = buf.Write(fn)
	_, _ = buf.Write(r.Header().AcceptedVersions().Bytes())
	if opt.withToken {
		_, _ = buf.Write(r.Header().Token())
	}
	_, _ = buf.Write(pp)
	b := buf.Bytes()
	p = bytex.FromString(strconv.FormatUint(xxhash.Sum64(b), 16))
	bytebufferpool.Put(buf)
	return
}

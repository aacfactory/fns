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
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/json"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"strconv"
	"sync"
)

// +-------------------------------------------------------------------------------------------------------------------+

type Request interface {
	Fn() (service []byte, fn []byte)
	Header() (header Header)
	Argument() (argument Argument)
	Hash() (p []byte)
}

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

func WithAuthorization(authorization []byte) RequestOption {
	return func(options *RequestOptions) {
		options.header.authorization = authorization
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

func NewRequest(ctx context.Context, service []byte, fn []byte, arg Argument, options ...RequestOption) (v Request) {
	opt := &RequestOptions{
		header: Header{},
	}
	if options != nil && len(options) > 0 {
		for _, option := range options {
			option(opt)
		}
	}
	if arg == nil {
		arg = EmptyArgument()
	}
	if len(opt.header.processId) == 0 {
		opt.header.processId = uid.Bytes()
	}
	prev, hasPrev := tryLoadRequest(ctx)
	if hasPrev {
		header := prev.Header()
		if len(opt.header.requestId) == 0 {
			opt.header.requestId = header.requestId
		}
		if len(opt.header.deviceId) == 0 {
			opt.header.deviceId = header.deviceId
		}
		if len(opt.header.deviceIp) == 0 {
			opt.header.deviceIp = header.deviceIp
		}
		if len(opt.header.authorization) == 0 {
			opt.header.authorization = header.authorization
		}
		if len(opt.header.acceptedVersions) == 0 {
			opt.header.acceptedVersions = header.acceptedVersions
		}
		opt.header.internal = true
	}
	v = &request{
		header:   opt.header,
		service:  service,
		fn:       fn,
		argument: arg,
		hash:     nil,
		hashOnce: new(sync.Once),
	}
	return
}

type request struct {
	header   Header
	service  []byte
	fn       []byte
	argument Argument
	hash     []byte
	hashOnce *sync.Once
}

func (r *request) Fn() (service []byte, fn []byte) {
	service, fn = r.service, r.fn
	return
}

func (r *request) Header() (header Header) {
	header = r.header
	return
}

func (r *request) Argument() (argument Argument) {
	argument = r.argument
	return
}

func (r *request) Hash() (p []byte) {
	r.hashOnce.Do(func() {
		body, _ := json.Marshal(r.argument)
		buf := bytebufferpool.Get()
		_, _ = buf.Write(r.service)
		_, _ = buf.Write(r.fn)
		_, _ = buf.Write(r.header.AcceptedVersions().Bytes())
		_, _ = buf.Write(body)
		b := buf.Bytes()
		bytebufferpool.Put(buf)
		r.hash = bytex.FromString(strconv.FormatUint(xxhash.Sum64(b), 16))
	})
	p = r.hash
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	contextRequestKey = "@fns:services:request"
)

func tryLoadRequest(ctx context.Context) (r Request, has bool) {
	v := ctx.Value(contextRequestKey)
	if v == nil {
		return
	}
	r, has = v.(Request)
	return
}

func LoadRequest(ctx context.Context) Request {
	v := ctx.Value(contextRequestKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no service request in context")))
		return nil
	}
	r, ok := v.(Request)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: request in context is not github.com/aacfactory/fns/services.Request")))
		return nil
	}
	return r
}

func withRequest(ctx context.Context, r Request) context.Context {
	return context.WithValue(ctx, contextRequestKey, r)
}

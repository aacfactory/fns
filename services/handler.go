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

package services

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/avros"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/mmhash"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
	"golang.org/x/sync/singleflight"
	"strconv"
	"sync/atomic"
)

var (
	slashBytes = []byte{'/'}
)

var (
	ErrDeviceId               = errors.NotAcceptable("fns: X-Fns-Device-Id is required")
	ErrInvalidPath            = errors.Warning("fns: invalid path")
	ErrInvalidBody            = errors.Warning("fns: invalid body")
	ErrInvalidRequestVersions = errors.Warning("fns: invalid request versions")
)

func Handler(endpoints Endpoints) transports.MuxHandler {
	return &endpointsHandler{
		endpoints: endpoints,
		loaded:    atomic.Bool{},
		infos:     nil,
		group:     singleflight.Group{},
	}
}

type endpointsHandler struct {
	endpoints Endpoints
	loaded    atomic.Bool
	infos     EndpointInfos
	group     singleflight.Group
}

func (handler *endpointsHandler) Name() string {
	return "endpoints"
}

func (handler *endpointsHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *endpointsHandler) Match(_ context.Context, method []byte, path []byte, header transports.Header) bool {
	if !handler.loaded.Load() {
		handler.infos = handler.endpoints.Info()
		handler.loaded.Store(true)
	}
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		return false
	}

	ep := pathItems[1]
	fn := pathItems[2]
	endpoint, hasEndpoint := handler.infos.Find(ep)
	if !hasEndpoint {
		return false
	}
	if endpoint.Internal {
		return false
	}
	fi, hasFn := endpoint.Functions.Find(fn)
	if !hasFn {
		return false
	}
	if fi.Internal {
		return false
	}
	if fi.Readonly {
		return bytes.Equal(method, transports.MethodGet)
	}
	if !bytes.Equal(method, transports.MethodPost) {
		return false
	}
	ok := bytes.Equal(method, transports.MethodPost) &&
		(bytes.Equal(header.Get(transports.ContentTypeHeaderName), transports.ContentTypeJsonHeaderValue) ||
			bytes.Equal(header.Get(transports.ContentTypeHeaderName), transports.ContentTypeAvroHeaderValue))
	return ok
}

func (handler *endpointsHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	groupKeyBuf := bytebufferpool.Get()

	// path
	path := r.Path()
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		bytebufferpool.Put(groupKeyBuf)
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
		return
	}
	ep := pathItems[1]
	fn := pathItems[2]
	_, _ = groupKeyBuf.Write(path)

	// header >>>
	options := make([]RequestOption, 0, 1)
	// device id
	deviceId := r.Header().Get(transports.DeviceIdHeaderName)
	if len(deviceId) == 0 {
		bytebufferpool.Put(groupKeyBuf)
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	options = append(options, WithDeviceId(deviceId))
	_, _ = groupKeyBuf.Write(deviceId)
	// device ip
	deviceIp := transports.DeviceIp(r)
	if len(deviceIp) > 0 {
		options = append(options, WithDeviceIp(deviceIp))
	}
	// request id
	requestId := r.Header().Get(transports.RequestIdHeaderName)
	if len(requestId) > 0 {
		options = append(options, WithRequestId(requestId))
	}
	// request version
	acceptedVersions := r.Header().Get(transports.RequestVersionsHeaderName)
	if len(acceptedVersions) > 0 {
		intervals, intervalsErr := versions.ParseIntervals(acceptedVersions)
		if intervalsErr != nil {
			bytebufferpool.Put(groupKeyBuf)
			w.Failed(ErrInvalidRequestVersions.WithMeta("path", bytex.ToString(path)).WithMeta("versions", bytex.ToString(acceptedVersions)).WithCause(intervalsErr))
			return
		}
		options = append(options, WithRequestVersions(intervals))
		_, _ = groupKeyBuf.Write(acceptedVersions)
	}
	// authorization
	authorization := r.Header().Get(transports.AuthorizationHeaderName)
	if len(authorization) > 0 {
		options = append(options, WithToken(authorization))
		_, _ = groupKeyBuf.Write(authorization)
	}

	// header <<<

	// param
	var param objects.Object
	method := r.Method()
	if bytes.Equal(method, transports.MethodGet) {
		// query
		queryParams := r.Params()
		param = transports.ObjectParams(queryParams)
		_, _ = groupKeyBuf.Write(queryParams.Encode())
	} else {
		// body
		body, bodyErr := r.Body()
		if bodyErr != nil {
			bytebufferpool.Put(groupKeyBuf)
			w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
			return
		}
		contentType := r.Header().Get(transports.ContentTypeHeaderName)
		if bytes.Equal(contentType, transports.ContentTypeJsonHeaderValue) {
			param = json.RawMessage(body)
		} else if bytes.Equal(contentType, transports.ContentTypeAvroHeaderValue) {
			param = avros.RawMessage(body)
		} else {
			if json.Validate(body) {
				param = json.RawMessage(body)
			} else {
				bytebufferpool.Put(groupKeyBuf)
				w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
				return
			}
		}
		_, _ = groupKeyBuf.Write(body)
	}

	// handle
	groupKey := strconv.FormatUint(mmhash.Sum64(groupKeyBuf.Bytes()), 16)
	bytebufferpool.Put(groupKeyBuf)
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		v, err = handler.endpoints.Request(
			r, ep, fn,
			param,
			options...,
		)
		return
	})
	handler.group.Forget(groupKey)
	if err != nil {
		w.Failed(err)
		return
	}
	response := v.(Response)

	if response.Valid() {
		w.Succeed(response.Value())
	} else {
		w.Succeed(nil)
	}
}

type MuxHandler interface {
	transports.MuxHandler
	Services() []Service
}

type Middleware interface {
	transports.Middleware
	Services() []Service
}

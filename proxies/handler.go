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

package proxies

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/mmhash"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/bytebufferpool"
	"golang.org/x/sync/singleflight"
	"strconv"
)

var (
	slashBytes = []byte{'/'}
)

func NewProxyHandler(manager clusters.ClusterEndpointsManager, dialer transports.Dialer) transports.MuxHandler {
	return &proxyHandler{
		manager: manager,
		dialer:  dialer,
		group:   singleflight.Group{},
	}
}

type proxyHandler struct {
	manager clusters.ClusterEndpointsManager
	dialer  transports.Dialer
	group   singleflight.Group
}

func (handler *proxyHandler) Name() string {
	return "proxy"
}

func (handler *proxyHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *proxyHandler) Match(_ context.Context, method []byte, path []byte, header transports.Header) bool {
	if bytes.Equal(method, transports.MethodPost) {
		return len(bytes.Split(path, slashBytes)) == 3 &&
			(bytes.Equal(header.Get(transports.ContentTypeHeaderName), transports.ContentTypeJsonHeaderValue) ||
				bytes.Equal(header.Get(transports.ContentTypeHeaderName), transports.ContentTypeAvroHeaderValue))
	}
	if bytes.Equal(method, transports.MethodGet) {
		return len(bytes.Split(path, slashBytes)) == 3
	}
	return false
}

func (handler *proxyHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	groupKeyBuf := bytebufferpool.Get()
	// path
	path := r.Path()
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		bytebufferpool.Put(groupKeyBuf)
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
		return
	}
	service := pathItems[1]
	fn := pathItems[2]
	_, _ = groupKeyBuf.Write(path)
	// device id
	deviceId := r.Header().Get(transports.DeviceIdHeaderName)
	if len(deviceId) == 0 {
		bytebufferpool.Put(groupKeyBuf)
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	_, _ = groupKeyBuf.Write(deviceId)

	// discovery
	endpointGetOptions := make([]services.EndpointGetOption, 0, 1)
	var intervals versions.Intervals
	acceptedVersions := r.Header().Get(transports.RequestVersionsHeaderName)
	if len(acceptedVersions) > 0 {
		var intervalsErr error
		intervals, intervalsErr = versions.ParseIntervals(acceptedVersions)
		if intervalsErr != nil {
			bytebufferpool.Put(groupKeyBuf)
			w.Failed(ErrInvalidRequestVersions.WithMeta("path", bytex.ToString(path)).WithMeta("versions", bytex.ToString(acceptedVersions)).WithCause(intervalsErr))
			return
		}
		endpointGetOptions = append(endpointGetOptions, services.EndpointVersions(intervals))
		_, _ = groupKeyBuf.Write(acceptedVersions)
	}

	var queryParams transports.Params
	var body []byte
	method := r.Method()
	if bytes.Equal(method, transports.MethodGet) {
		queryParams = r.Params()
		queryParamsBytes := queryParams.Encode()
		path = append(path, '?')
		path = append(path, queryParamsBytes...)
		_, _ = groupKeyBuf.Write(queryParamsBytes)
	} else {
		var bodyErr error
		body, bodyErr = r.Body()
		if bodyErr != nil {
			bytebufferpool.Put(groupKeyBuf)
			w.Failed(errors.Warning("fns: read request body failed").WithCause(bodyErr).
				WithMeta("endpoint", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn)))
			return
		}
		_, _ = groupKeyBuf.Write(body)
	}

	groupKey := strconv.FormatUint(mmhash.Sum64(groupKeyBuf.Bytes()), 16)
	bytebufferpool.Put(groupKeyBuf)
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		address, internal, has := handler.manager.FnAddress(r, service, fn, endpointGetOptions...)
		if !has {
			err = errors.NotFound("fns: endpoint was not found").
				WithMeta("endpoint", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn))
			return
		}
		if internal {
			err = errors.NotFound("fns: fn was internal").
				WithMeta("endpoint", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn))
			return
		}

		client, clientErr := handler.dialer.Dial(bytex.FromString(address))
		if clientErr != nil {
			err = errors.Warning("fns: dial endpoint failed").WithCause(clientErr).
				WithMeta("endpoint", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn))
			return
		}
		status, respHeader, respBody, doErr := client.Do(r, method, path, r.Header(), body)
		if doErr != nil {
			err = errors.Warning("fns: send request to endpoint failed").WithCause(doErr).
				WithMeta("endpoint", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn))
			return
		}
		v = Response{
			Status: status,
			Header: respHeader,
			Value:  respBody,
		}
		return
	})

	if err != nil {
		w.Failed(err)
		return
	}

	response := v.(Response)
	if response.Header.Len() > 0 {
		response.Header.Foreach(func(key []byte, values [][]byte) {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		})
	}
	w.SetStatus(response.Status)
	_, _ = w.Write(response.Value)
}

type Response struct {
	Status int
	Header transports.Header
	Value  []byte
}

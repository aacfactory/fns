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

package clusters

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/avro"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/avros"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/window"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/fns/transports/middlewares/compress"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"sync/atomic"
)

type Fn struct {
	log          logs.Logger
	address      string
	endpointName string
	name         string
	internal     bool
	readonly     bool
	path         []byte
	signature    signatures.Signature
	errs         *window.Times
	health       atomic.Bool
	client       transports.Client
}

func (fn *Fn) Enable() bool {
	return fn.health.Load()
}

func (fn *Fn) Name() string {
	return fn.name
}

func (fn *Fn) Internal() bool {
	return fn.internal
}

func (fn *Fn) Readonly() bool {
	return fn.readonly
}

func (fn *Fn) Handle(ctx services.Request) (v interface{}, err error) {
	if !ctx.Header().Internal() {
		err = errors.Warning("fns: request must be internal")
		return
	}
	// header >>>
	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	// try copy transport request header
	transportRequestHeader, hasTransportRequestHeader := transports.TryLoadRequestHeader(ctx)
	if hasTransportRequestHeader {
		transportRequestHeader.Foreach(func(key []byte, values [][]byte) {
			ok := bytes.Equal(key, transports.CookieHeaderName) ||
				bytes.Equal(key, transports.XForwardedForHeaderName) ||
				bytes.Equal(key, transports.OriginHeaderName) ||
				bytes.Index(key, transports.UserHeaderNamePrefix) == 0
			if ok {
				for _, value := range values {
					header.Add(key, value)
				}
			}
		})
	}
	// accept
	header.Set(transports.AcceptEncodingHeaderName, transports.ContentTypeAvroHeaderValue)
	// content-type
	header.Set(transports.ContentTypeHeaderName, internalContentTypeHeader)

	// endpoint id
	endpointId := ctx.Header().EndpointId()
	if len(endpointId) > 0 {
		header.Set(transports.EndpointIdHeaderName, endpointId)
	}
	// device id
	deviceId := ctx.Header().DeviceId()
	if len(deviceId) > 0 {
		header.Set(transports.DeviceIdHeaderName, deviceId)
	}
	// device ip
	deviceIp := ctx.Header().DeviceIp()
	if len(deviceIp) > 0 {
		header.Set(transports.DeviceIpHeaderName, deviceIp)
	}
	// request id
	requestId := ctx.Header().RequestId()
	if len(requestId) > 0 {
		header.Set(transports.RequestIdHeaderName, requestId)
	}
	// request version
	requestVersion := ctx.Header().AcceptedVersions()
	if len(requestVersion) > 0 {
		header.Set(transports.RequestVersionsHeaderName, requestVersion.Bytes())
	}
	// authorization
	token := ctx.Header().Token()
	if len(token) > 0 {
		header.Set(transports.AuthorizationHeaderName, token)
	}
	// header <<<

	// body
	userValues := make([]Entry, 0, 1)
	ctx.UserValues(func(key []byte, val any) {
		p, encodeErr := json.Marshal(val)
		if encodeErr != nil {
			return
		}
		userValues = append(userValues, Entry{
			Key:   key,
			Value: p,
		})
	})
	var argument []byte
	var argumentErr error
	if ctx.Param().Valid() {
		param := ctx.Param().Value()
		argument, argumentErr = avro.Marshal(param)
	}
	if argumentErr != nil {
		err = errors.Warning("fns: encode request argument failed").WithCause(argumentErr).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
		return
	}
	rb := RequestBody{
		ContextUserValues: userValues,
		Params:            argument,
	}
	body, bodyErr := avro.Marshal(rb)
	if bodyErr != nil {
		err = errors.Warning("fns: encode body failed").WithCause(bodyErr).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
		return
	}
	// sign
	signature := fn.signature.Sign(body)
	header.Set(transports.SignatureHeaderName, signature)

	// do
	status, respHeader, respBody, doErr := fn.client.Do(ctx, transports.MethodPost, fn.path, header, body)
	if doErr != nil {
		n := fn.errs.Incr()
		if n > 10 {
			fn.health.Store(false)
		}
		err = errors.Warning("fns: internal endpoint handle failed").WithCause(doErr).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
		return
	}
	// debug
	if fn.log.DebugEnabled() {
		fn.log.Debug().
			With("address", fn.address).
			With("endpoint", fn.endpointName).
			With("fn", fn.name).
			With("status", status).
			Message(fmt.Sprintf("fns: status of internal endpoint is %d", status))
	}

	// try copy transport response header
	transportResponseHeader, hasTransportResponseHeader := transports.TryLoadResponseHeader(ctx)
	if hasTransportResponseHeader {
		respHeader.Foreach(func(key []byte, values [][]byte) {
			ok := bytes.Equal(key, transports.CookieHeaderName) ||
				bytes.Index(key, transports.UserHeaderNamePrefix) == 0
			if ok {
				for _, value := range values {
					transportResponseHeader.Add(key, value)
				}
			}
		})
	}

	if status == 200 {
		if fn.errs.Value() > 0 {
			fn.errs.Decr()
		}
		respBody, err = compress.DecodeResponse(respHeader, respBody)
		if err != nil {
			err = errors.Warning("fns: internal endpoint handle failed").WithCause(err).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
			return
		}
		rsb := ResponseBody{}
		decodeErr := avro.Unmarshal(respBody, &rsb)
		if decodeErr != nil {
			err = errors.Warning("fns: internal endpoint handle failed").WithCause(decodeErr).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
			return
		}
		trace, hasTrace := tracings.Load(ctx)
		if hasTrace {
			span, hasSpan := rsb.GetSpan()
			if hasSpan {
				trace.Mount(span)
			}
		}
		if rsb.Succeed {
			v = avros.RawMessage(rsb.Data)
		} else {
			codeErr := &errors.CodeErrorImpl{}
			_ = avro.Unmarshal(rsb.Data, codeErr)
			err = codeErr
		}
		return
	}
	switch status {
	case http.StatusServiceUnavailable:
		fn.health.Store(false)
		err = ErrUnavailable
		break
	case http.StatusTooManyRequests:
		err = ErrTooMayRequest
		break
	case http.StatusTooEarly:
		err = ErrTooEarly
		break
	case 555:
		respBody, err = compress.DecodeResponse(respHeader, respBody)
		if err != nil {
			err = errors.Warning("fns: internal endpoint handle failed").WithCause(err).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
			return
		}
		codeErr := &errors.CodeErrorImpl{}
		_ = avro.Unmarshal(respBody, codeErr)
		err = codeErr
		break
	case 666:
		respBody, err = compress.DecodeResponse(respHeader, respBody)
		if err != nil {
			err = errors.Warning("fns: internal endpoint handle failed").WithCause(err).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
			return
		}
		err = errors.New(666, "***INTERNAL FAILED***", string(respBody))
		break
	}
	return
}

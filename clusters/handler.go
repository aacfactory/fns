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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/fns/commons/protos"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"google.golang.org/protobuf/proto"
)

var (
	slashBytes          = []byte{'/'}
	internalContentType = []byte("application/proto+fns")
	spanAttachmentKey   = []byte("span")
)

const (
	jsonEncoding int64 = iota + 1
	protoEncoding
)

func NewInternalHandler(local services.Endpoints, signature signatures.Signature) transports.MuxHandler {
	return &InternalHandler{
		signature: signature,
		endpoints: local,
	}
}

type InternalHandler struct {
	signature signatures.Signature
	endpoints services.Endpoints
}

func (handler *InternalHandler) Name() string {
	return "internal"
}

func (handler *InternalHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *InternalHandler) Match(_ context.Context, method []byte, path []byte, header transports.Header) bool {
	matched := bytes.Equal(method, transports.MethodPost) &&
		len(bytes.Split(path, slashBytes)) == 3 &&
		bytes.Equal(header.Get(transports.ContentTypeHeaderName), internalContentType) &&
		len(header.Get(transports.SignatureHeaderName)) != 0
	return matched
}

func (handler *InternalHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	// path
	path := r.Path()
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
		return
	}
	service := pathItems[1]
	fn := pathItems[2]

	// sign
	sign := r.Header().Get(transports.SignatureHeaderName)
	if len(sign) == 0 {
		w.Failed(ErrSignatureLost.WithMeta("path", bytex.ToString(path)))
		return
	}
	// body
	body, bodyErr := r.Body()
	if bodyErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
		return
	}

	if !handler.signature.Verify(body, sign) {
		w.Failed(ErrSignatureUnverified.WithMeta("path", bytex.ToString(path)))
		return
	}

	rb := RequestBody{}
	decodeErr := proto.Unmarshal(body, &rb)
	if decodeErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)).WithCause(decodeErr))
		return
	}
	// user values
	for _, userValue := range rb.ContextUserValues {
		r.SetUserValue(userValue.Key, userValue.Value)
	}

	// header >>>
	options := make([]services.RequestOption, 0, 1)
	// internal
	options = append(options, services.WithInternalRequest())
	// endpoint id
	endpointId := r.Header().Get(transports.EndpointIdHeaderName)
	if len(endpointId) > 0 {
		options = append(options, services.WithEndpointId(endpointId))
	}
	// device id
	deviceId := r.Header().Get(transports.DeviceIdHeaderName)
	if len(deviceId) == 0 {
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	options = append(options, services.WithDeviceId(deviceId))
	// device ip
	deviceIp := r.Header().Get(transports.DeviceIpHeaderName)
	if len(deviceIp) > 0 {
		options = append(options, services.WithDeviceIp(deviceIp))
	}
	// request id
	requestId := r.Header().Get(transports.RequestIdHeaderName)
	hasRequestId := len(requestId) > 0
	if hasRequestId {
		options = append(options, services.WithRequestId(requestId))
	}
	// request version
	acceptedVersions := r.Header().Get(transports.RequestVersionsHeaderName)
	if len(acceptedVersions) > 0 {
		intervals, intervalsErr := versions.ParseIntervals(acceptedVersions)
		if intervalsErr != nil {
			w.Failed(ErrInvalidRequestVersions.WithMeta("path", bytex.ToString(path)).WithMeta("versions", bytex.ToString(acceptedVersions)).WithCause(intervalsErr))
			return
		}
		options = append(options, services.WithRequestVersions(intervals))
	}
	// authorization
	authorization := r.Header().Get(transports.AuthorizationHeaderName)
	if len(authorization) > 0 {
		options = append(options, services.WithToken(authorization))
	}
	// header <<<

	// param
	var param objects.Object
	if rb.Encoding == jsonEncoding {
		param = json.RawMessage(rb.Params)
	} else if rb.Encoding == protoEncoding {
		param = protos.RawMessage(rb.Params)
	}

	var ctx context.Context = r

	// handle
	response, err := handler.endpoints.Request(
		ctx, service, fn,
		param,
		options...,
	)
	succeed := err == nil
	encoding := jsonEncoding
	var data []byte
	var dataErr error
	var span *tracings.Span
	if succeed {
		if response.Valid() {
			responseValue := response.Value()
			msg, isMsg := responseValue.(proto.Message)
			if isMsg {
				encoding = protoEncoding
				data, dataErr = proto.Marshal(msg)
			} else {
				data, dataErr = json.Marshal(responseValue)
			}
		} else {
			data = json.NullBytes
		}
	} else {
		data, dataErr = json.Marshal(errors.Wrap(err))
	}
	if dataErr != nil {
		encoding = jsonEncoding
		succeed = false
		data, _ = json.Marshal(errors.Warning("fns: encode endpoint response failed").WithMeta("path", bytex.ToString(path)).WithCause(dataErr))
	}

	if hasRequestId {
		trace, hasTrace := tracings.Load(ctx)
		if hasTrace {
			span = trace.Span
		}
	}

	rsb := ResponseBody{
		Succeed:     succeed,
		Encoding:    encoding,
		Data:        data,
		Attachments: make([]*Entry, 0, 1),
	}
	if span != nil {
		rsb.Span = span
	}

	p, encodeErr := proto.Marshal(&rsb)
	if encodeErr != nil {
		w.Failed(errors.Warning("fns: proto marshal failed").WithCause(encodeErr))
		return
	}
	w.Header().Set(transports.ContentTypeHeaderName, internalContentType)
	_, _ = w.Write(p)
}

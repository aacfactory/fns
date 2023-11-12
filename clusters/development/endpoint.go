package development

import (
	"bytes"
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/services/tracing"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net/http"
)

func NewEndpoint(name []byte, internal bool, document documents.Document, client transports.Client, signature signatures.Signature) services.Endpoint {
	return &Endpoint{
		name:      bytex.ToString(name),
		internal:  internal,
		document:  document,
		client:    client,
		signature: signature,
	}
}

type Endpoint struct {
	name      string
	internal  bool
	document  documents.Document
	client    transports.Client
	signature signatures.Signature
}

func (endpoint *Endpoint) Name() (name string) {
	name = endpoint.name
	return
}

func (endpoint *Endpoint) Internal() (ok bool) {
	ok = endpoint.internal
	return
}

func (endpoint *Endpoint) Document() (document documents.Document) {
	document = endpoint.document
	return
}

func (endpoint *Endpoint) Handle(ctx services.Request) (v interface{}, err error) {
	// header >>>
	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	// try copy transport request header
	transportRequestHeader, hasTransportRequestHeader := transports.TryLoadRequestHeader(ctx)
	if hasTransportRequestHeader {
		transportRequestHeader.Foreach(func(key []byte, values [][]byte) {
			for _, value := range values {
				header.Add(key, value)
			}
		})
	}
	// content-type
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// endpoint id
	endpointId := ctx.Header().EndpointId()
	if len(endpointId) > 0 {
		header.Set(bytex.FromString(transports.EndpointIdHeaderName), endpointId)
	}
	// device id
	deviceId := ctx.Header().DeviceId()
	if len(deviceId) > 0 {
		header.Set(bytex.FromString(transports.DeviceIdHeaderName), deviceId)
	}
	// device ip
	deviceIp := ctx.Header().DeviceIp()
	if len(deviceIp) > 0 {
		header.Set(bytex.FromString(transports.DeviceIpHeaderName), deviceIp)
	}
	// request id
	requestId := ctx.Header().RequestId()
	if len(requestId) > 0 {
		header.Set(bytex.FromString(transports.RequestIdHeaderName), requestId)
	}
	// request version
	requestVersion := ctx.Header().AcceptedVersions()
	if len(requestVersion) > 0 {
		header.Set(bytex.FromString(transports.RequestVersionsHeaderName), requestVersion.Bytes())
	}
	// authorization
	token := ctx.Header().Token()
	if len(token) > 0 {
		header.Set(bytex.FromString(transports.AuthorizationHeaderName), token)
	}
	// header <<<

	// path
	service, fn := ctx.Fn()
	sln := len(service)
	fln := len(fn)
	path := make([]byte, sln+fln+2)
	path[0] = '/'
	path[sln+1] = '/'
	copy(path[1:], service)
	copy(path[sln+2:], fn)

	// body
	userValues := make([]Entry, 0, 1)
	ctx.UserValues(func(key []byte, val any) {
		p, encodeErr := json.Marshal(val)
		if encodeErr != nil {
			return
		}
		userValues = append(userValues, Entry{
			Key: key,
			Val: p,
		})
	})
	argument, argumentErr := ctx.Argument().MarshalJSON()
	if argumentErr != nil {
		err = errors.Warning("fns: encode request argument failed").WithCause(argumentErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}
	rb := RequestBody{
		UserValues: userValues,
		Argument:   argument,
	}
	body, bodyErr := json.Marshal(rb)
	if bodyErr != nil {
		err = errors.Warning("fns: encode body failed").WithCause(bodyErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}
	// sign
	signature := endpoint.signature.Sign(body)
	header.Set(bytex.FromString(transports.SignatureHeaderName), signature)

	// do
	status, _, respBody, doErr := endpoint.client.Do(ctx, methodPost, path, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development endpoint handle failed").WithCause(doErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}

	if status == 200 {
		rsb := ResponseBody{}
		decodeErr := json.Unmarshal(respBody, &rsb)
		if decodeErr != nil {
			err = errors.Warning("fns: development endpoint handle failed").WithCause(decodeErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
			return
		}
		if rsb.Span != nil && len(rsb.Span.Id) > 0 {
			tracing.MountSpan(ctx, rsb.Span)
		}
		if rsb.Succeed {
			v = rsb.Data
		} else {
			err = errors.Decode(rsb.Data)
		}
		return
	}
	switch status {
	case http.StatusServiceUnavailable:
		err = ErrUnavailable
		break
	case http.StatusTooManyRequests:
		err = ErrTooMayRequest
		break
	case http.StatusTooEarly:
		err = ErrTooEarly
		break
	}
	return
}

func (endpoint *Endpoint) Shutdown(_ context.Context) {
	endpoint.client.Close()
}

// +-------------------------------------------------------------------------------------------------------------------+

type Entry struct {
	Key []byte          `json:"key"`
	Val json.RawMessage `json:"val"`
}

type RequestBody struct {
	UserValues []Entry         `json:"userValues"`
	Argument   json.RawMessage `json:"argument"`
}

type ResponseBody struct {
	Succeed bool            `json:"succeed"`
	Data    json.RawMessage `json:"data"`
	Span    *tracing.Span   `json:"span"`
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewEndpointsHandler(signature signatures.Signature) transports.Handler {
	return &EndpointsHandler{
		signature: signature,
	}
}

type EndpointsHandler struct {
	signature signatures.Signature
}

func (handler *EndpointsHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	// path
	path := r.Path()
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
		return
	}
	service := pathItems[1]
	fn := pathItems[2]

	// body
	body, bodyErr := r.Body()
	if bodyErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
		return
	}

	rb := RequestBody{}
	decodeRequestBodyErr := json.Unmarshal(body, &rb)
	if decodeRequestBodyErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)).WithCause(decodeRequestBodyErr))
		return
	}
	// user values
	for _, userValue := range rb.UserValues {
		r.SetUserValue(userValue.Key, userValue.Val)
	}
	// header >>>
	options := make([]services.RequestOption, 0, 1)
	// internal
	options = append(options, services.WithInternalRequest())
	// endpoint id
	endpointId := r.Header().Get(bytex.FromString(transports.EndpointIdHeaderName))
	if len(endpointId) > 0 {
		options = append(options, services.WithEndpointId(endpointId))
	}
	// device id
	deviceId := r.Header().Get(bytex.FromString(transports.DeviceIdHeaderName))
	if len(deviceId) == 0 {
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	options = append(options, services.WithDeviceId(deviceId))
	// device ip
	deviceIp := r.Header().Get(bytex.FromString(transports.DeviceIpHeaderName))
	if len(deviceIp) > 0 {
		options = append(options, services.WithDeviceIp(deviceIp))
	}
	// request id
	requestId := r.Header().Get(bytex.FromString(transports.RequestIdHeaderName))
	hasRequestId := len(requestId) > 0
	if hasRequestId {
		options = append(options, services.WithRequestId(requestId))
	}
	// request version
	acceptedVersions := r.Header().Get(bytex.FromString(transports.RequestVersionsHeaderName))
	if len(acceptedVersions) > 0 {
		intervals, intervalsErr := versions.ParseIntervals(acceptedVersions)
		if intervalsErr != nil {
			w.Failed(ErrInvalidRequestVersions.WithMeta("path", bytex.ToString(path)).WithMeta("versions", bytex.ToString(acceptedVersions)).WithCause(intervalsErr))
			return
		}
		options = append(options, services.WithRequestVersions(intervals))
	}
	// authorization
	authorization := r.Header().Get(bytex.FromString(transports.AuthorizationHeaderName))
	if len(authorization) > 0 {
		options = append(options, services.WithToken(authorization))
	}

	// header <<<

	var ctx context.Context = r

	// runtime
	rt := runtime.Load(r)

	// handle
	response, err := rt.Endpoints().Request(
		ctx, service, fn,
		services.NewArgument(rb.Argument),
		options...,
	)
	succeed := err == nil
	var data []byte
	var dataErr error
	var span *tracing.Span

	if succeed {
		if response.Exist() {
			data, dataErr = json.Marshal(response)
		}
	} else {
		data, dataErr = json.Marshal(errors.Map(err))
	}
	if dataErr != nil {
		succeed = false
		data, _ = json.Marshal(errors.Warning("fns: encode endpoint response failed").WithMeta("path", bytex.ToString(path)).WithCause(dataErr))
	}

	if hasRequestId {
		span = tracing.LoadSpan(ctx)
	}

	rsb := ResponseBody{
		Succeed: succeed,
		Data:    data,
		Span:    span,
	}

	w.Succeed(rsb)
	return
}

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
)

func NewEndpoint(name []byte, internal bool, document *documents.Document, client transports.Client) services.Endpoint {
	return &Endpoint{
		name:     bytex.ToString(name),
		internal: internal,
		document: document,
		client:   client,
	}
}

type Endpoint struct {
	name     string
	internal bool
	document *documents.Document
	client   transports.Client
}

func (endpoint *Endpoint) Name() (name string) {
	//TODO implement me
	panic("implement me")
}

func (endpoint *Endpoint) Internal() (ok bool) {
	//TODO implement me
	panic("implement me")
}

func (endpoint *Endpoint) Document() (document *documents.Document) {
	//TODO implement me
	panic("implement me")
}

func (endpoint *Endpoint) Handle(ctx services.Request) (v interface{}, err error) {
	//TODO implement me
	panic("implement me")
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
	// sign
	sign := r.Header().Get(bytex.FromString(transports.SignatureHeaderName))
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

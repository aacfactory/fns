package clusters

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
)

var (
	slashBytes          = []byte{'/'}
	internalContentType = bytex.FromString("application/json+fns")
)

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
	Span    *tracings.Span  `json:"span"`
}

// NewInternalHandler
// endpoints: local endpoints
func NewInternalHandler(id string, signature signatures.Signature, endpoints services.Endpoints) transports.MuxHandler {
	return &InternalHandler{
		id:        bytex.FromString(id),
		signature: signature,
		endpoints: endpoints,
	}
}

type InternalHandler struct {
	id        []byte
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
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), internalContentType) &&
		len(header.Get(bytex.FromString(transports.SignatureHeaderName))) != 0 &&
		bytes.Equal(header.Get(bytex.FromString(transports.EndpointIdHeaderName)), handler.id)
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
	decodeErr := json.Unmarshal(body, &rb)
	if decodeErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)).WithCause(decodeErr))
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
	if len(endpointId) == 0 {
		endpointId = handler.id
	}
	options = append(options, services.WithEndpointId(endpointId))
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

	// handle
	response, err := handler.endpoints.Request(
		ctx, service, fn,
		rb.Argument,
		options...,
	)
	succeed := err == nil
	var data []byte
	var dataErr error
	var span *tracings.Span
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
		trace, hasTrace := tracings.Load(ctx)
		if hasTrace {
			span = trace.Span
		}
	}

	rsb := ResponseBody{
		Succeed: succeed,
		Data:    data,
		Span:    span,
	}

	w.Succeed(rsb)
}

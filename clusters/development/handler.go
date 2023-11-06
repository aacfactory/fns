package development

import (
	"bytes"
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/tracing"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
)

var (
	ErrDeviceId               = errors.New(http.StatusNotAcceptable, "***NOT ACCEPTABLE**", "fns: X-Fns-Device-Id is required")
	ErrInvalidPath            = errors.Warning("fns: invalid path")
	ErrInvalidBody            = errors.Warning("fns: invalid body")
	ErrInvalidRequestVersions = errors.Warning("fns: invalid request versions")
	ErrSignatureLost          = errors.New(488, "***SIGNATURE LOST***", "X-Fns-Signature was required")
	ErrSignatureUnverified    = errors.New(458, "***SIGNATURE INVALID***", "X-Fns-Signature was invalid")
)

var (
	methodPost = bytex.FromString(http.MethodPost)
)

var (
	slashBytes = []byte{'/'}
)

var (
	contentType = bytex.FromString("application/json+dev")
)

func NewHandler(signature signatures.Signature) transports.MuxHandler {
	return &Handler{
		signature: signature,
	}
}

type Handler struct {
	log       logs.Logger
	signature signatures.Signature
}

func (handler *Handler) Name() string {
	return "development"
}

func (handler *Handler) Construct(options transports.MuxHandlerOptions) error {
	handler.log = options.Log
	return nil
}

func (handler *Handler) Match(method []byte, path []byte, header transports.Header) bool {
	ok := bytes.Equal(method, methodPost) &&
		len(bytes.Split(path, slashBytes)) == 3 &&
		len(header.Get(bytex.FromString(transports.SignatureHeaderName))) != 0 &&
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), contentType)
	return ok
}

func (handler *Handler) Handle(w transports.ResponseWriter, r transports.Request) {
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
	rb := clusters.RequestBody{}
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

	rsb := clusters.ResponseBody{
		Succeed: succeed,
		Data:    data,
		Span:    span,
	}

	w.Succeed(rsb)
	return
}

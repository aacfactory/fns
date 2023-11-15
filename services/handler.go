package services

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net/http"
)

var (
	slashBytes = []byte{'/'}
)

var (
	ErrDeviceId               = errors.New(http.StatusNotAcceptable, "***NOT ACCEPTABLE**", "fns: X-Fns-Device-Id is required")
	ErrInvalidPath            = errors.Warning("fns: invalid path")
	ErrInvalidBody            = errors.Warning("fns: invalid body")
	ErrInvalidRequestVersions = errors.Warning("fns: invalid request versions")
)

func Handler(endpoints EndpointInfos) transports.MuxHandler {
	return &endpointsHandler{
		endpoints: endpoints,
	}
}

type endpointsHandler struct {
	endpoints EndpointInfos
}

func (handler *endpointsHandler) Name() string {
	return "endpoints"
}

func (handler *endpointsHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *endpointsHandler) Match(method []byte, path []byte, header transports.Header) bool {
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		return false
	}
	ep := pathItems[1]
	fn := pathItems[2]
	endpoint, hasEndpoint := handler.endpoints.Find(ep)
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
		ok := bytes.Equal(method, transports.MethodGet)
		return ok
	}
	ok := bytes.Equal(method, transports.MethodPost) && bytex.ToString(header.Get(bytex.FromString(transports.ContentTypeHeaderName))) == transports.ContentTypeJsonHeaderValue
	return ok
}

func (handler *endpointsHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	rt := runtime.Load(r)
	// path
	path := r.Path()
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
		return
	}
	ep := pathItems[1]
	fn := pathItems[2]
	var param scanner.Scanner
	if bytes.Equal(r.Method(), transports.MethodGet) {
		param = transports.ParamsScanner(r.Params())
	} else {
		// body
		body, bodyErr := r.Body()
		if bodyErr != nil {
			w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
			return
		}
		param = json.RawMessage(body)
	}

	// header >>>
	options := make([]RequestOption, 0, 1)
	// device id
	deviceId := r.Header().Get(bytex.FromString(transports.DeviceIdHeaderName))
	if len(deviceId) == 0 {
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	options = append(options, WithDeviceId(deviceId))
	// device ip
	deviceIp := transports.DeviceIp(r)
	if len(deviceIp) > 0 {
		options = append(options, WithDeviceIp(deviceIp))
	}
	// request id
	requestId := r.Header().Get(bytex.FromString(transports.RequestIdHeaderName))
	if len(requestId) > 0 {
		options = append(options, WithRequestId(requestId))
	}
	// request version
	acceptedVersions := r.Header().Get(bytex.FromString(transports.RequestVersionsHeaderName))
	if len(acceptedVersions) > 0 {
		intervals, intervalsErr := versions.ParseIntervals(acceptedVersions)
		if intervalsErr != nil {
			w.Failed(ErrInvalidRequestVersions.WithMeta("path", bytex.ToString(path)).WithMeta("versions", bytex.ToString(acceptedVersions)).WithCause(intervalsErr))
			return
		}
		options = append(options, WithRequestVersions(intervals))
	}
	// authorization
	authorization := r.Header().Get(bytex.FromString(transports.AuthorizationHeaderName))
	if len(authorization) > 0 {
		options = append(options, WithToken(authorization))
	}

	// header <<<

	// handle
	response, err := rt.Endpoints().Request(
		r, ep, fn,
		param,
		options...,
	)
	if err != nil {
		w.Failed(err)
		return
	}
	if response.Exist() {
		w.Succeed(response)
	} else {
		w.Succeed(nil)
	}
}

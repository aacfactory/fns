package handlers

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"net/http"
)

var (
	methodGet  = bytex.FromString(http.MethodGet)
	methodPost = bytex.FromString(http.MethodPost)
)

var (
	slashBytes = []byte{'/'}
)

func NewEndpointsHandler(endpoints services.HostEndpoints) transports.MuxHandler {
	return &EndpointsHandler{
		endpoints: endpoints,
	}
}

type EndpointsHandler struct {
	// todo matcher via documents
	endpoints services.HostEndpoints
}

func (handler *EndpointsHandler) Name() string {
	return "endpoints"
}

func (handler *EndpointsHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *EndpointsHandler) Match(method []byte, path []byte, header transports.Header) bool {
	// todo use matcher
	ok := bytes.Equal(method, methodPost) &&
		len(bytes.Split(path, slashBytes)) == 3 &&
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), bytex.FromString(transports.ContentTypeJsonHeaderValue))
	return ok
}

func (handler *EndpointsHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	rt := runtime.Load(r)
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

	// header >>>
	options := make([]services.RequestOption, 0, 1)
	// device id
	deviceId := r.Header().Get(bytex.FromString(transports.DeviceIdHeaderName))
	if len(deviceId) == 0 {
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	options = append(options, services.WithDeviceId(deviceId))
	// device ip
	deviceIp := transports.DeviceIp(r)
	if len(deviceIp) > 0 {
		options = append(options, services.WithDeviceIp(deviceIp))
	}
	// request id
	requestId := r.Header().Get(bytex.FromString(transports.RequestIdHeaderName))
	if len(requestId) > 0 {
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

	// handle
	response, err := rt.Endpoints().Request(
		r, service, fn,
		services.NewArgument(body),
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

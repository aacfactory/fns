package proxies

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"net/http"
)

var (
	methodPost = bytex.FromString(http.MethodPost)
)

var (
	slashBytes = []byte{'/'}
	commaBytes = []byte{','}
)

func NewProxyHandler() transports.MuxHandler {
	return &ProxyHandler{}
}

type ProxyHandler struct {
}

func (handler *ProxyHandler) Name() string {
	return "proxy"
}

func (handler *ProxyHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *ProxyHandler) Match(method []byte, path []byte, header transports.Header) bool {
	ok := bytes.Equal(method, methodPost) &&
		len(bytes.Split(path, slashBytes)) == 3 &&
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), bytex.FromString(transports.ContentTypeJsonHeaderValue))
	return ok
}

func (handler *ProxyHandler) Handle(w transports.ResponseWriter, r transports.Request) {
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
	// device id
	deviceId := r.Header().Get(bytex.FromString(transports.DeviceIdHeaderName))
	if len(deviceId) == 0 {
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	// discovery
	discoveryOptions := make([]services.EndpointGetOption, 0, 1)
	var intervals versions.Intervals
	acceptedVersions := r.Header().Get(bytex.FromString(transports.RequestVersionsHeaderName))
	if len(acceptedVersions) > 0 {
		var intervalsErr error
		intervals, intervalsErr = versions.ParseIntervals(acceptedVersions)
		if intervalsErr != nil {
			w.Failed(ErrInvalidRequestVersions.WithMeta("path", bytex.ToString(path)).WithMeta("versions", bytex.ToString(acceptedVersions)).WithCause(intervalsErr))
			return
		}
		discoveryOptions = append(discoveryOptions, services.EndpointVersions(intervals))
	}
	endpoint, has := rt.Discovery().Get(r, service, discoveryOptions...)
	if !has {
		w.Failed(errors.NotFound("fns: endpoint was not found").
			WithMeta("service", bytex.ToString(service)).
			WithMeta("fn", bytex.ToString(fn)))
		return
	}

	// body
	body, bodyErr := r.Body()
	if bodyErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
		return
	}

	// request
	request := services.AcquireRequest(r, service, fn, services.NewArgument(body))

	// dispatch
	v, handleErr := endpoint.Handle(request)
	if handleErr != nil {
		services.ReleaseRequest(request)
		w.Failed(handleErr)
		return
	}
	services.ReleaseRequest(request)
	w.Succeed(v)
}

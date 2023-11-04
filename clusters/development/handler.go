package development

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"net/http"
)

var (
	proxyDevContentType     = bytex.FromString("application/json+dev")
	sharedProxyEndpointName = "shared"
)

var (
	ErrDeviceId               = errors.New(http.StatusNotAcceptable, "***NOT ACCEPTABLE**", "fns: X-Fns-Device-Id is required")
	ErrInvalidPath            = errors.Warning("fns: invalid path")
	ErrInvalidBody            = errors.Warning("fns: invalid body")
	ErrInvalidRequestVersions = errors.Warning("fns: invalid request versions")
)

var (
	methodPost = bytex.FromString(http.MethodPost)
)

var (
	slashBytes = []byte{'/'}
)

type ProxyHandlerConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

func NewProxyHandler() transports.MuxHandler {
	return &ProxyHandler{}
}

type ProxyHandler struct {
	enabled bool
}

func (handler *ProxyHandler) Name() string {
	return "development"
}

func (handler *ProxyHandler) Construct(options transports.MuxHandlerOptions) error {
	config := ProxyHandlerConfig{}
	err := options.Config.As(&config)
	if err != nil {
		err = errors.Warning("fns: construct development handler failed").WithCause(err)
		return err
	}
	handler.enabled = config.Enabled
	return nil
}

func (handler *ProxyHandler) Match(method []byte, path []byte, header transports.Header) bool {
	ok := handler.enabled && bytes.Equal(method, methodPost) &&
		len(bytes.Split(path, slashBytes)) == 3 &&
		len(header.Get(bytex.FromString(transports.SignatureHeaderName))) != 0 &&
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), proxyDevContentType)
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
	// body
	body, bodyErr := r.Body()
	if bodyErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
		return
	}
	// sign todo

	if string(service) == sharedProxyEndpointName {
		v, err := handler.handleSharedProxy(rt.Shared(), fn, body)
		if err != nil {
			w.Failed(err)
			return
		}
		w.Succeed(v)
		return
	}
	v, err := handler.handleProxy(rt.Discovery(), service, fn, body, r.Header())
	if err != nil {
		w.Failed(err)
		return
	}
	w.Succeed(v)
}

func (handler *ProxyHandler) handleSharedProxy(shared shareds.Shared, fn []byte, body []byte) (v interface{}, err error) {

	return
}

func (handler *ProxyHandler) handleProxy(discovery services.Discovery, service []byte, fn []byte, body []byte, header transports.Header) (v interface{}, err error) {
	// todo same as internal handler
	// internal request and internal response

	return
}

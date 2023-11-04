package proxies

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
)

var (
	proxyDevContentType     = bytex.FromString("application/json+dev")
	sharedProxyEndpointName = "shared"
)

type DevelopmentHandlerConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

func NewDevelopmentHandler() transports.MuxHandler {
	return &DevelopmentHandler{}
}

type DevelopmentHandler struct {
	enabled bool
}

func (handler *DevelopmentHandler) Name() string {
	return "development"
}

func (handler *DevelopmentHandler) Construct(options transports.MuxHandlerOptions) error {
	config := DevelopmentHandlerConfig{}
	err := options.Config.As(&config)
	if err != nil {
		err = errors.Warning("fns: construct development handler failed").WithCause(err)
		return err
	}
	handler.enabled = config.Enabled
	return nil
}

func (handler *DevelopmentHandler) Match(method []byte, path []byte, header transports.Header) bool {
	ok := handler.enabled && bytes.Equal(method, methodPost) &&
		len(bytes.Split(path, slashBytes)) == 3 &&
		len(header.Get(bytex.FromString(transports.SignatureHeaderName))) != 0 &&
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), proxyDevContentType)
	return ok
}

func (handler *DevelopmentHandler) Handle(w transports.ResponseWriter, r transports.Request) {
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

func (handler *DevelopmentHandler) handleSharedProxy(shared shareds.Shared, fn []byte, body []byte) (v interface{}, err error) {

	return
}

func (handler *DevelopmentHandler) handleProxy(discovery services.Discovery, service []byte, fn []byte, body []byte, header transports.Header) (v interface{}, err error) {
	// todo same as internal handler
	// internal request and internal response

	return
}

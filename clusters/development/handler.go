package development

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"net/http"
)

var (
	ErrTooEarly               = errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: service is not ready, try later again")
	ErrUnavailable            = errors.Unavailable("fns: service is closed")
	ErrTooMayRequest          = errors.New(http.StatusTooManyRequests, "***TOO MANY REQUEST***", "fns: too may request, try again later.")
	ErrDeviceId               = errors.New(http.StatusNotAcceptable, "***NOT ACCEPTABLE**", "fns: X-Fns-Device-Id is required")
	ErrInvalidPath            = errors.Warning("fns: invalid path")
	ErrInvalidBody            = errors.Warning("fns: invalid body")
	ErrInvalidRequestVersions = errors.Warning("fns: invalid request versions")
	ErrSignatureLost          = errors.New(488, "***SIGNATURE LOST***", "X-Fns-Signature was required")
	ErrSignatureUnverified    = errors.New(458, "***SIGNATURE INVALID***", "X-Fns-Signature was invalid")
)

var (
	contentType           = bytex.FromString("application/json+dev")
	methodPost            = bytex.FromString(http.MethodPost)
	slashBytes            = []byte{'/'}
	developmentPathPrefix = []byte("/development/")
)

func NewHandler(signature signatures.Signature, discovery services.Discovery) transports.MuxHandler {
	return &Handler{
		log:       nil,
		signature: signature,
		discovery: NewDiscoveryHandler(signature, discovery),
		shared:    NewSharedHandler(signature),
		endpoints: NewEndpointsHandler(signature),
	}
}

type Handler struct {
	log       logs.Logger
	signature signatures.Signature
	discovery transports.Handler
	shared    transports.Handler
	endpoints transports.Handler
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
		(len(bytes.Split(path, slashBytes)) == 3 || bytes.Index(path, developmentPathPrefix) == 0) &&
		len(header.Get(bytex.FromString(transports.SignatureHeaderName))) != 0 &&
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), contentType)
	return ok
}

func (handler *Handler) Handle(w transports.ResponseWriter, r transports.Request) {
	path := r.Path()
	// sign
	sign := r.Header().Get(bytex.FromString(transports.SignatureHeaderName))
	if len(sign) == 0 {
		w.Failed(ErrSignatureLost.WithMeta("path", bytex.ToString(path)))
		return
	}
	// match
	if bytes.Index(path, discoveryHandlePathPrefix) == 0 {
		handler.discovery.Handle(w, r)
	} else if bytes.Index(path, shardHandlePathPrefix) == 0 {
		handler.shared.Handle(w, r)
	} else {
		handler.endpoints.Handle(w, r)
	}
}

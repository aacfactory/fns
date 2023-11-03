package handlers

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"sync"
)

// NewOpenapiHandler
// todo move into contrib with viewer
func NewOpenapiHandler() transports.MuxHandler {
	return &OpenapiHandler{
		doc:  nil,
		once: new(sync.Once),
	}
}

type OpenapiHandler struct {
	doc  json.RawMessage
	once *sync.Once
}

func (handler *OpenapiHandler) Name() string {
	return "openapi"
}

func (handler *OpenapiHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *OpenapiHandler) Match(method []byte, path []byte, _ transports.Header) bool {
	ok := bytes.Equal(method, methodGet) && bytes.Equal(path, bytex.FromString(documents.ServicesOpenapiPath))
	return ok
}

func (handler *OpenapiHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	handler.once.Do(func() {
		rt := runtime.Load(r)
		doc := rt.Endpoints().Documents()
		api := doc.Openapi("")
		p, _ := json.Marshal(api)
		handler.doc = p
	})
	w.Succeed(handler.doc)
}

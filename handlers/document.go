package handlers

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"sync"
)

var (
	documentsHandlerPath = bytex.FromString("/services/documents")
)

type DocumentHandlerConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

func NewDocumentHandler() transports.MuxHandler {
	return &DocumentsHandler{}
}

type DocumentsHandler struct {
	enabled bool
	doc     json.RawMessage
	once    *sync.Once
}

func (handler *DocumentsHandler) Name() string {
	return "documents"
}

func (handler *DocumentsHandler) Construct(options transports.MuxHandlerOptions) error {
	config := DocumentHandlerConfig{}
	err := options.Config.As(&config)
	if err != nil {
		err = errors.Warning("fns: construct document handler failed").WithCause(err)
		return err
	}
	handler.enabled = config.Enabled
	if handler.enabled {
		handler.once = new(sync.Once)
	}
	return nil
}

func (handler *DocumentsHandler) Match(method []byte, path []byte, _ transports.Header) bool {
	ok := handler.enabled && bytes.Equal(method, methodGet) && bytes.Equal(path, documentsHandlerPath)
	return ok
}

func (handler *DocumentsHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	handler.once.Do(func() {
		rt := runtime.Load(r)
		host, ok := rt.Endpoints().(services.HostEndpoints)
		if ok {
			doc := host.Documents()
			p, _ := json.Marshal(doc)
			handler.doc = p
		}
	})
	w.Succeed(handler.doc)
}

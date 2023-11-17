package transport

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"sync"
)

var (
	documentsHandlerPath = bytex.FromString("/documents")
)

type DocumentHandlerConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

func DocumentHandler() transports.MuxHandler {
	return &documentHandler{}
}

type documentHandler struct {
	enabled  bool
	document json.RawMessage
	once     *sync.Once
}

func (handler *documentHandler) Name() string {
	return "documents"
}

func (handler *documentHandler) Construct(options transports.MuxHandlerOptions) error {
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

func (handler *documentHandler) Match(method []byte, path []byte, _ transports.Header) bool {
	ok := handler.enabled && bytes.Equal(method, transports.MethodGet) && bytes.Equal(path, documentsHandlerPath)
	return ok
}

func (handler *documentHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	handler.once.Do(func() {
		rt := runtime.Load(r)
		id := rt.AppId()
		infos := rt.Endpoints().Info()
		for _, info := range infos {
			if info.Id == id {
				p, _ := json.Marshal(info.Document)
				handler.document = p
				break
			}
		}
	})
	w.Succeed(handler.document)
}

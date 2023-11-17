package documents

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

type HandlerConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

func Handler() transports.MuxHandler {
	return &handler{}
}

type handler struct {
	enabled  bool
	document json.RawMessage
	once     *sync.Once
}

func (handler *handler) Name() string {
	return "documents"
}

func (handler *handler) Construct(options transports.MuxHandlerOptions) error {
	config := HandlerConfig{}
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

func (handler *handler) Match(method []byte, path []byte, _ transports.Header) bool {
	ok := handler.enabled && bytes.Equal(method, transports.MethodGet) && bytes.Equal(path, documentsHandlerPath)
	return ok
}

func (handler *handler) Handle(w transports.ResponseWriter, r transports.Request) {
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

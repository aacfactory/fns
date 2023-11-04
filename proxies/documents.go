package proxies

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
	"golang.org/x/sync/singleflight"
	"net/http"
)

const (
	documentsHandlerName = "documents"
)

var (
	documentsHandlerPath = bytex.FromString("/services/documents")
)

var (
	methodGet = bytex.FromString(http.MethodGet)
)

type DocumentHandlerConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

func NewDocumentsHandler() transports.MuxHandler {
	return &DocumentsHandler{}
}

type DocumentsHandler struct {
	enabled bool
	group   *singleflight.Group
}

func (handler *DocumentsHandler) Name() string {
	return documentsHandlerName
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
		handler.group = new(singleflight.Group)
	}
	return nil
}

func (handler *DocumentsHandler) Match(method []byte, path []byte, _ transports.Header) bool {
	ok := handler.enabled && bytes.Equal(method, methodGet) && bytes.Equal(path, documentsHandlerPath)
	return ok
}

func (handler *DocumentsHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	v, _, _ := handler.group.Do(documentsHandlerName, func() (v interface{}, err error) {
		rt := runtime.Load(r)
		infos := rt.Discovery().Endpoints(r)
		docs := make(documents.VersionSortedDocuments, 0, 1)
		for _, info := range infos {
			docs = docs.Add(info.Id, info.Document)
		}
		v = docs
		return
	})
	w.Succeed(v)
}

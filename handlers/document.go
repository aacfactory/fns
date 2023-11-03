package handlers

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"sync"
)

func FetchDocuments(ctx context.Context, client transports.Client) (v *documents.Documents, err error) {
	status, _, body, doErr := client.Do(ctx, methodGet, bytex.FromString(documents.ServicesDocumentsPath), nil, nil)
	if doErr != nil {
		err = errors.Warning("fns: fetch endpoints document failed").WithCause(doErr)
		return
	}
	if status == 200 {
		if len(body) == 0 {
			return
		}
		v = &documents.Documents{}
		err = json.Unmarshal(body, v)
		if err != nil {
			err = errors.Warning("fns: fetch endpoints document failed").WithCause(err)
			return
		}
		return
	}
	err = errors.Warning("fns: fetch endpoints document failed").WithCause(errors.Decode(body)).WithMeta("status", fmt.Sprintf("%d", status))
	return
}

func NewDocumentHandler() transports.MuxHandler {
	return &DocumentsHandler{
		doc:  nil,
		once: new(sync.Once),
	}
}

type DocumentsHandler struct {
	doc  json.RawMessage
	once *sync.Once
}

func (handler *DocumentsHandler) Name() string {
	return "documents"
}

func (handler *DocumentsHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *DocumentsHandler) Match(method []byte, path []byte, _ transports.Header) bool {
	ok := bytes.Equal(method, methodGet) && bytes.Equal(path, bytex.FromString(documents.ServicesDocumentsPath))
	return ok
}

func (handler *DocumentsHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	handler.once.Do(func() {
		rt := runtime.Load(r)
		doc := rt.Endpoints().Documents()
		p, _ := json.Marshal(doc)
		handler.doc = p
	})
	w.Succeed(handler.doc)
}

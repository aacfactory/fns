package clusters

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services/tracing"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"golang.org/x/sync/singleflight"
	"net/http"
)

var (
	slashBytes  = []byte{'/'}
	methodPost  = bytex.FromString(http.MethodPost)
	contentType = bytex.FromString("application/json+fns")
)

type Entry struct {
	Key string          `json:"key"`
	Val json.RawMessage `json:"val"`
}

type RequestBody struct {
	UserValues []Entry         `json:"userValues"`
	Argument   json.RawMessage `json:"argument"`
}

type ResponseBody struct {
	Succeed bool            `json:"succeed"`
	Data    json.RawMessage `json:"data"`
	Span    *tracing.Span   `json:"span"`
}

type InternalHandler struct {
	rt        *runtime.Runtime
	group     *singleflight.Group
	signature signatures.Signature
}

func (handler *InternalHandler) Match(method []byte, path []byte, header transports.Header) bool {
	matched := bytes.Equal(method, methodPost) &&
		len(bytes.Split(path, slashBytes)) == 3 &&
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), contentType) &&
		len(header.Get(bytex.FromString(transports.RequestInternalHeaderName))) != 0 &&
		len(header.Get(bytex.FromString(transports.SignatureHeaderName))) != 0
	return matched
}

func (handler *InternalHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	//TODO implement me
	panic("implement me")
}

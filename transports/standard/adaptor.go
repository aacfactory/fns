package standard

import (
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"net/http"
	"sync"
)

var (
	requestPool  = sync.Pool{}
	responsePool = sync.Pool{}
)

func HttpTransportHandlerAdaptor(h transports.Handler, maxRequestBody int) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := context.Acquire(request.Context())

		var r *Request
		cr := requestPool.Get()
		if cr == nil {
			r = new(Request)
		} else {
			r = cr.(*Request)
		}
		r.Context = ctx
		r.maxBodySize = maxRequestBody
		r.request = request

		var w *responseWriter
		cw := responsePool.Get()
		if cw == nil {
			w = new(responseWriter)
		} else {
			w = cw.(*responseWriter)
		}
		w.Context = ctx
		w.writer = writer
		w.header = WrapHttpHeader(writer.Header())
		w.result = transports.AcquireResultResponseWriter()

		h.Handle(w, r)

		writer.WriteHeader(w.Status())

		if bodyLen := w.BodyLen(); bodyLen > 0 {
			body := w.Body()
			n := 0
			for n < bodyLen {
				nn, writeErr := writer.Write(body[n:])
				if writeErr != nil {
					break
				}
				n += nn
			}
		}

		transports.ReleaseResultResponseWriter(w.result)
		w.Context = nil
		w.writer = nil
		w.header = nil
		w.result = nil
		w.hijacked = false
		responsePool.Put(w)

		r.Context = nil
		r.maxBodySize = 0
		r.request = nil
		requestPool.Put(r)

		context.Release(ctx)
	})
}

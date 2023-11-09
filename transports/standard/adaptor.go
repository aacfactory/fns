package standard

import (
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/bytebufferpool"
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

		buf := bytebufferpool.Get()

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
		w.body = buf

		h.Handle(w, r)

		writer.WriteHeader(w.Status())

		bodyLen := buf.Len()
		if bodyLen > 0 {
			body := buf.Bytes()
			n := 0
			for n < bodyLen {
				nn, writeErr := writer.Write(body[n:])
				if writeErr != nil {
					break
				}
				n += nn
			}
		}

		w.Context = nil
		w.writer = nil
		w.status = 0
		w.header = nil
		w.body = nil
		w.hijacked = false
		responsePool.Put(w)

		bytebufferpool.Put(buf)

		r.Context = nil
		r.maxBodySize = 0
		r.request = nil
		requestPool.Put(r)

		context.Release(ctx)
	})
}

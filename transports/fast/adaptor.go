package fast

import (
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"sync"
)

var (
	requestPool  = sync.Pool{}
	responsePool = sync.Pool{}
)

func handlerAdaptor(h transports.Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var r *Request
		cr := requestPool.Get()
		if cr == nil {
			r = new(Request)
		} else {
			r = cr.(*Request)
		}
		r.ctx = ctx

		buf := bytebufferpool.Get()

		var w *responseWriter
		cw := responsePool.Get()
		if cw == nil {
			w = new(responseWriter)
		} else {
			w = cw.(*responseWriter)
		}
		w.ctx = ctx
		w.header = ResponseHeader{
			ResponseHeader: &ctx.Response.Header,
		}
		w.body = buf

		h.Handle(w, r)

		ctx.SetStatusCode(w.Status())

		if bodyLen := buf.Len(); bodyLen > 0 {
			body := buf.Bytes()
			n := 0
			for n < bodyLen {
				nn, writeErr := ctx.Write(body[n:])
				if writeErr != nil {
					break
				}
				n += nn
			}
		}

		w.ctx = nil
		w.status = 0
		w.header = nil
		w.body = nil
		responsePool.Put(w)

		bytebufferpool.Put(buf)

		r.ctx = nil
		requestPool.Put(r)
	}
}

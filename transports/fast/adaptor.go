package fast

import (
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
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
			r.locals = make(context.Entries, 0, 1)
		} else {
			r = cr.(*Request)
		}
		r.ctx = ctx

		var w *responseWriter
		cw := responsePool.Get()
		if cw == nil {
			w = new(responseWriter)
			w.locals = make(context.Entries, 0, 1)
		} else {
			w = cw.(*responseWriter)
		}
		w.ctx = ctx
		w.result = transports.AcquireResultResponseWriter()

		h.Handle(w, r)

		ctx.SetStatusCode(w.Status())

		if bodyLen := w.BodyLen(); bodyLen > 0 {
			body := w.Body()
			n := 0
			for n < bodyLen {
				nn, writeErr := ctx.Write(body[n:])
				if writeErr != nil {
					break
				}
				n += nn
			}
		}

		transports.ReleaseResultResponseWriter(w.result)
		w.ctx = nil
		w.locals.Reset()
		w.result = nil
		responsePool.Put(w)

		r.ctx = nil
		r.locals.Reset()
		requestPool.Put(r)
	}
}

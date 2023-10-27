package fast

import (
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
)

func handlerAdaptor(h transports.Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r := &Request{
			ctx: ctx,
		}

		buf := bytebufferpool.Get()
		w := convertFastHttpRequestCtxToResponseWriter(ctx, buf)

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

		bytebufferpool.Put(buf)
	}
}

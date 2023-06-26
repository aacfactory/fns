package fast

import (
	"bufio"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"net"
	"net/http"
	"strconv"
)

func handlerAdaptor(h transports.Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		r, convertErr := convertFastHttpRequestCtxToRequest(ctx)
		if convertErr != nil {
			p, _ := json.Marshal(errors.Warning("fns: fasthttp handler adapt failed ").WithCause(convertErr))
			ctx.Response.Reset()
			ctx.SetStatusCode(555)
			ctx.SetContentTypeBytes(bytex.FromString(transports.ContentTypeJsonHeaderValue))
			ctx.SetBody(p)
			return
		}

		buf := bytebufferpool.Get()
		w := convertFastHttpRequestCtxToResponseWriter(ctx, buf)

		h.Handle(w, r)

		for k, vv := range w.Header() {
			for _, v := range vv {
				ctx.Response.Header.Add(k, v)
			}
		}

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

func convertFastHttpRequestCtxToRequest(ctx *fasthttp.RequestCtx) (r *transports.Request, err error) {
	r, err = transports.NewRequest(ctx, ctx.Method(), ctx.RequestURI())
	if err != nil {
		err = errors.Warning("fns: convert fasthttp request to transport request failed").WithCause(err)
		return
	}

	r.SetHost(ctx.Host())

	if ctx.IsTLS() {
		r.UseTLS()
		r.SetTLSConnectionState(ctx.TLSConnectionState())
	}

	r.SetProto(ctx.Request.Header.Protocol())
	r.SetRemoteAddr(bytex.FromString(ctx.RemoteAddr().String()))

	ctx.Request.Header.VisitAll(func(key, value []byte) {
		sk := bytex.ToString(key)
		sv := bytex.ToString(value)
		r.Header().Set(sk, sv)
	})

	if ctx.IsPost() || ctx.IsPut() {
		r.SetBody(ctx.PostBody())
	}

	return
}

func convertFastHttpRequestCtxToResponseWriter(ctx *fasthttp.RequestCtx, writer transports.WriteBuffer) (w transports.ResponseWriter) {
	w = &responseWriter{
		ctx:    ctx,
		status: 0,
		header: make(transports.Header),
		body:   writer,
	}
	return
}

type responseWriter struct {
	ctx    *fasthttp.RequestCtx
	status int
	header transports.Header
	body   transports.WriteBuffer
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) SetStatus(status int) {
	w.status = status
}

func (w *responseWriter) Header() transports.Header {
	return w.header
}

func (w *responseWriter) Succeed(v interface{}) {
	if v == nil {
		w.status = http.StatusOK
		return
	}
	body, encodeErr := json.Marshal(v)
	if encodeErr != nil {
		w.Failed(errors.Warning("fns: transport write succeed result failed").WithCause(encodeErr))
		return
	}

	w.status = http.StatusOK

	bodyLen := len(body)
	if bodyLen > 0 {
		w.Header().Set(transports.ContentLengthHeaderName, strconv.Itoa(bodyLen))
		w.Header().Set(transports.ContentTypeHeaderName, transports.ContentTypeJsonHeaderValue)
		w.write(body, bodyLen)
	}
	return
}

func (w *responseWriter) Failed(cause errors.CodeError) {
	if cause == nil {
		cause = errors.Warning("fns: error is lost")
	}
	body, encodeErr := json.Marshal(cause)
	if encodeErr != nil {
		body = []byte(`{"message": "fns: transport write failed result failed"}`)
		return
	}
	w.status = cause.Code()
	bodyLen := len(body)
	if bodyLen > 0 {
		w.Header().Set(transports.ContentLengthHeaderName, strconv.Itoa(bodyLen))
		w.Header().Set(transports.ContentTypeHeaderName, transports.ContentTypeJsonHeaderValue)
		w.write(body, bodyLen)
	}
	return
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if body == nil {
		return 0, nil
	}
	bodyLen := len(body)
	w.write(body, bodyLen)
	return bodyLen, nil
}

func (w *responseWriter) Body() []byte {
	return w.body.Bytes()
}

func (w *responseWriter) write(body []byte, bodyLen int) {
	n := 0
	for n < bodyLen {
		nn, writeErr := w.body.Write(body[n:])
		if writeErr != nil {
			break
		}
		n += nn
	}
	return
}

func (w *responseWriter) Hijack(f func(conn net.Conn, rw *bufio.ReadWriter) (err error)) (async bool, err error) {
	if f == nil {
		err = errors.Warning("fns: hijack function is nil")
		return
	}
	w.ctx.Hijack(func(c net.Conn) {
		_ = f(c, nil)
	})
	async = true
	return
}

func (w *responseWriter) Hijacked() bool {
	return w.ctx.Hijacked()
}

package fast

import (
	"bufio"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/fasthttp"
	"net"
	"strconv"
	"time"
)

type responseWriter struct {
	ctx    *fasthttp.RequestCtx
	result *transports.ResultResponseWriter
}

func (w *responseWriter) Deadline() (time.Time, bool) {
	return w.ctx.Deadline()
}

func (w *responseWriter) Done() <-chan struct{} {
	return w.ctx.Done()
}

func (w *responseWriter) Err() error {
	return w.ctx.Err()
}

func (w *responseWriter) Value(key any) any {
	return w.ctx.Value(key)
}

func (w *responseWriter) UserValue(key []byte) (val any) {
	return w.ctx.UserValueBytes(key)
}

func (w *responseWriter) ScanUserValue(key []byte, val any) (has bool, err error) {
	v := w.UserValue(key)
	if v == nil {
		return
	}
	s := scanner.New(v)
	err = s.Scan(val)
	if err != nil {
		err = errors.Warning("fns: scan context value failed").WithMeta("key", bytex.ToString(key)).WithCause(err)
		return
	}
	has = true
	return
}

func (w *responseWriter) SetUserValue(key []byte, val any) {
	w.ctx.SetUserValueBytes(key, val)
}

func (w *responseWriter) UserValues(fn func(key []byte, val any)) {
	w.ctx.VisitUserValues(fn)
}

func (w *responseWriter) Status() int {
	return w.result.Status()
}

func (w *responseWriter) SetStatus(status int) {
	w.result.SetStatus(status)
}

func (w *responseWriter) SetCookie(cookie *transports.Cookie) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)
	c.SetKeyBytes(cookie.Key())
	c.SetValueBytes(cookie.Value())
	c.SetExpire(cookie.Expire())
	c.SetMaxAge(cookie.MaxAge())
	c.SetDomainBytes(cookie.Domain())
	c.SetPathBytes(cookie.Path())
	c.SetHTTPOnly(cookie.HTTPOnly())
	c.SetSecure(cookie.Secure())
	c.SetSameSite(fasthttp.CookieSameSite(cookie.SameSite()))
	w.ctx.Response.Header.SetCookie(c)
}

func (w *responseWriter) Header() transports.Header {
	return ResponseHeader{
		&w.ctx.Response.Header,
	}
}

func (w *responseWriter) Succeed(v interface{}) {
	w.result.Succeed(v)
	if bodyLen := w.result.BodyLen(); bodyLen > 0 {
		w.Header().Set(bytex.FromString(transports.ContentLengthHeaderName), bytex.FromString(strconv.Itoa(bodyLen)))
		w.Header().Set(bytex.FromString(transports.ContentTypeHeaderName), bytex.FromString(transports.ContentTypeJsonHeaderValue))
	}
	return
}

func (w *responseWriter) Failed(cause error) {
	w.result.Failed(cause)
	if bodyLen := w.result.BodyLen(); bodyLen > 0 {
		w.Header().Set(bytex.FromString(transports.ContentLengthHeaderName), bytex.FromString(strconv.Itoa(bodyLen)))
		w.Header().Set(bytex.FromString(transports.ContentTypeHeaderName), bytex.FromString(transports.ContentTypeJsonHeaderValue))
	}
	return
}

func (w *responseWriter) Write(body []byte) (int, error) {
	return w.result.Write(body)
}

func (w *responseWriter) Body() []byte {
	return w.result.Body()
}

func (w *responseWriter) BodyLen() int {
	return w.result.BodyLen()
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

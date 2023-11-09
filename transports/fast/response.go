package fast

import (
	"bufio"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/valyala/fasthttp"
	"net"
	"net/http"
	"strconv"
	"time"
)

type responseWriter struct {
	ctx    *fasthttp.RequestCtx
	status int
	body   transports.WriteBuffer
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
	s := objects.NewScanner(v)
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
	return w.status
}

func (w *responseWriter) SetStatus(status int) {
	w.status = status
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
		w.Header().Set(bytex.FromString(transports.ContentLengthHeaderName), bytex.FromString(strconv.Itoa(bodyLen)))
		w.Header().Set(bytex.FromString(transports.ContentTypeHeaderName), bytex.FromString(transports.ContentTypeJsonHeaderValue))
		w.write(body, bodyLen)
	}
	return
}

func (w *responseWriter) Failed(cause error) {
	if cause == nil {
		cause = errors.Warning("fns: error is lost")
	}
	err := errors.Map(cause)
	body, encodeErr := json.Marshal(err)
	if encodeErr != nil {
		body = []byte(`{"message": "fns: transport write failed result failed"}`)
		return
	}
	w.status = err.Code()
	bodyLen := len(body)
	if bodyLen > 0 {
		w.Header().Set(bytex.FromString(transports.ContentLengthHeaderName), bytex.FromString(strconv.Itoa(bodyLen)))
		w.Header().Set(bytex.FromString(transports.ContentTypeHeaderName), bytex.FromString(transports.ContentTypeJsonHeaderValue))
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

package fast

import (
	"crypto/tls"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/fasthttp"
	"time"
)

type Request struct {
	ctx    *fasthttp.RequestCtx
	locals context.Entries
}

func (r *Request) Deadline() (time.Time, bool) {
	return r.ctx.Deadline()
}

func (r *Request) Done() <-chan struct{} {
	return r.ctx.Done()
}

func (r *Request) Err() error {
	return r.ctx.Err()
}

func (r *Request) Value(key any) any {
	return r.ctx.Value(key)
}

func (r *Request) UserValue(key []byte) (val any) {
	return r.ctx.UserValueBytes(key)
}

func (r *Request) ScanUserValue(key []byte, val any) (has bool, err error) {
	v := r.UserValue(key)
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

func (r *Request) SetUserValue(key []byte, val any) {
	r.ctx.SetUserValueBytes(key, val)
}

func (r *Request) UserValues(fn func(key []byte, val any)) {
	r.ctx.VisitUserValues(fn)
}

func (r *Request) LocalValue(key []byte) any {
	v := r.locals.Get(key)
	if v != nil {
		return v
	}
	return nil
}

func (r *Request) ScanLocalValue(key []byte, val any) (has bool, err error) {
	v := r.LocalValue(key)
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

func (r *Request) SetLocalValue(key []byte, val any) {
	r.locals.Set(key, val)
}

func (r *Request) TLS() bool {
	return r.ctx.IsTLS()
}

func (r *Request) TLSConnectionState() *tls.ConnectionState {
	return r.ctx.TLSConnectionState()
}

func (r *Request) RemoteAddr() []byte {
	return bytex.FromString(r.ctx.RemoteAddr().String())
}

func (r *Request) Proto() []byte {
	return r.ctx.Request.Header.Protocol()
}

func (r *Request) Host() []byte {
	return r.ctx.Host()
}

func (r *Request) Method() []byte {
	return r.ctx.Method()
}

func (r *Request) SetMethod(method []byte) {
	r.ctx.Request.Header.SetMethodBytes(method)
}

func (r *Request) Cookie(key []byte) (value []byte) {
	value = r.ctx.Request.Header.CookieBytes(key)
	return
}

func (r *Request) SetCookie(key []byte, value []byte) {
	r.ctx.Request.Header.SetCookieBytesKV(key, value)
}

func (r *Request) Header() transports.Header {
	return RequestHeader{&r.ctx.Request.Header}
}

func (r *Request) Path() []byte {
	return r.ctx.URI().Path()
}

func (r *Request) Params() transports.Params {
	return &Params{args: r.ctx.QueryArgs()}
}

func (r *Request) Body() ([]byte, error) {
	return r.ctx.PostBody(), nil
}

func (r *Request) SetBody(body []byte) {
	r.ctx.Request.SetBody(body)
}

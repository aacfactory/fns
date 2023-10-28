package fast

import (
	"crypto/tls"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/fasthttp"
	"time"
)

type Request struct {
	ctx *fasthttp.RequestCtx
}

func (r *Request) UserValue(key []byte) (val any) {
	return r.ctx.UserValueBytes(key)
}

func (r *Request) SetUserValue(key []byte, val any) {
	r.ctx.SetUserValueBytes(key, val)
}

func (r *Request) RemoveUserValue(key []byte) {
	r.ctx.RemoveUserValueBytes(key)
}

func (r *Request) ForeachUserValues(fn func(key []byte, val any)) {
	r.ctx.VisitUserValues(fn)
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

func (r *Request) Header() transports.Header {
	return &RequestHeader{&r.ctx.Request.Header}
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

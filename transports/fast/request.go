package fast

import (
	"context"
	"crypto/tls"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/fasthttp"
)

type Request struct {
	ctx *fasthttp.RequestCtx
}

func (r *Request) Context() context.Context {
	return r.ctx
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

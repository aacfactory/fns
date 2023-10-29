package standard

import (
	"crypto/tls"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/bytebufferpool"
	"io"
	"net/http"
	"time"
)

const (
	securedSchema = "https"
)

type Request struct {
	ctx         context.Context
	maxBodySize int
	request     *http.Request
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

func (r *Request) UserValue(key []byte) any {
	return r.ctx.UserValue(key)
}

func (r *Request) SetUserValue(key []byte, val any) {
	r.ctx.SetUserValue(key, val)
}

func (r *Request) UserValues(fn func(key []byte, val any)) {
	r.ctx.UserValues(fn)
}

func (r *Request) TLS() bool {
	return r.request.URL.Scheme == securedSchema
}

func (r *Request) TLSConnectionState() *tls.ConnectionState {
	return r.request.TLS
}

func (r *Request) RemoteAddr() []byte {
	return bytex.FromString(r.request.RemoteAddr)
}

func (r *Request) Proto() []byte {
	return bytex.FromString(r.request.Proto)
}

func (r *Request) Host() []byte {
	return bytex.FromString(r.request.Host)
}

func (r *Request) Method() []byte {
	return bytex.FromString(r.request.Method)
}

func (r *Request) Header() transports.Header {
	return transports.WrapHttpHeader(r.request.Header)
}

func (r *Request) Path() []byte {
	return bytex.FromString(r.request.URL.Path)
}

func (r *Request) Params() transports.Params {
	return &Params{
		values: r.request.URL.Query(),
	}
}

func (r *Request) Body() ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	b := bytex.Acquire4KBuffer()
	defer bytex.Release4KBuffer(b)
	for {
		n, readErr := r.request.Body.Read(b)
		if n > 0 {
			_, _ = buf.Write(b[0:n])
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, errors.Warning("fns: read request body failed").WithCause(readErr)
		}
		if r.maxBodySize > 0 {
			if buf.Len() > r.maxBodySize {
				return nil, errors.Warning("fns: read request body failed").WithCause(transports.ErrTooBigRequestBody)
			}
		}
	}
	return buf.Bytes(), nil
}

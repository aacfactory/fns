package standard

import (
	"bufio"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net"
	"net/http"
	"strconv"
)

func convertHttpResponseWriterToResponseWriter(ctx context.Context, w http.ResponseWriter, buf transports.WriteBuffer) transports.ResponseWriter {
	return &responseWriter{
		Context:  ctx,
		writer:   w,
		status:   0,
		header:   WrapHttpHeader(w.Header()),
		body:     buf,
		hijacked: false,
	}
}

type responseWriter struct {
	context.Context
	writer   http.ResponseWriter
	status   int
	header   transports.Header
	body     transports.WriteBuffer
	hijacked bool
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) SetStatus(status int) {
	w.status = status
}

func (w *responseWriter) SetCookie(cookie *transports.Cookie) {
	c := http.Cookie{
		Name:       bytex.ToString(cookie.Key()),
		Value:      bytex.ToString(cookie.Value()),
		Path:       bytex.ToString(cookie.Path()),
		Domain:     bytex.ToString(cookie.Domain()),
		Expires:    cookie.Expire(),
		RawExpires: "",
		MaxAge:     cookie.MaxAge(),
		Secure:     cookie.Secure(),
		HttpOnly:   cookie.HTTPOnly(),
		SameSite:   http.SameSite(cookie.SameSite()),
		Raw:        "",
		Unparsed:   nil,
	}
	http.SetCookie(w.writer, &c)
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
	h, ok := w.writer.(http.Hijacker)
	if !ok {
		err = errors.Warning("fns: connection can not be hijacked")
		return
	}
	conn, brw, hijackErr := h.Hijack()
	if hijackErr != nil {
		err = errors.Warning("fns: connection hijack failed").WithCause(hijackErr)
		return
	}
	w.hijacked = true
	err = f(conn, brw)
	return
}

func (w *responseWriter) Hijacked() bool {
	return w.hijacked
}

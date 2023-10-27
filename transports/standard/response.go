package standard

import (
	"bufio"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net"
	"net/http"
	"strconv"
)

func convertHttpResponseWriterToResponseWriter(w http.ResponseWriter, buf transports.WriteBuffer) transports.ResponseWriter {
	return &responseWriter{
		writer:   w,
		status:   0,
		header:   transports.WrapHttpHeader(w.Header()),
		body:     buf,
		hijacked: false,
	}
}

type responseWriter struct {
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

package standard

import (
	"bufio"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
	"io"
	"net"
	"net/http"
	"strconv"
)

func HttpTransportHandlerAdaptor(h transports.Handler, maxRequestBody int) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		r, convertErr := convertHttpRequestToRequest(request, maxRequestBody)
		if convertErr != nil {
			p, _ := json.Marshal(errors.Warning("fns: http handler adapt failed ").WithCause(convertErr))
			writer.Header().Set(transports.ContentTypeHeaderName, transports.ContentTypeJsonHeaderValue)
			writer.WriteHeader(555)
			_, _ = writer.Write(p)
			return
		}

		buf := bytebufferpool.Get()
		w := convertHttpResponseWriterToResponseWriter(writer, buf)

		h.Handle(w, r)

		writer.WriteHeader(w.Status())

		bodyLen := buf.Len()
		if bodyLen > 0 {
			body := buf.Bytes()
			n := 0
			for n < bodyLen {
				nn, writeErr := writer.Write(body[n:])
				if writeErr != nil {
					break
				}
				n += nn
			}
		}

		bytebufferpool.Put(buf)
	})
}

func convertHttpRequestToRequest(req *http.Request, bodyLimit int) (r *transports.Request, err error) {
	r, err = transports.NewRequestWithHeader(req.Context(), bytex.FromString(req.Method), bytex.FromString(req.RequestURI), transports.WrapHttpHeader(req.Header))
	if err != nil {
		err = errors.Warning("fns: new transport request from http request failed").WithCause(err)
		return
	}
	if req.URL.Scheme == "https" {
		r.UseTLS()
		r.SetTLSConnectionState(req.TLS)
	}
	r.SetHost(bytex.FromString(req.Host))
	r.SetRemoteAddr(bytex.FromString(req.RemoteAddr))
	r.SetProto(bytex.FromString(req.Proto))

	params := req.URL.Query()
	if params != nil && len(params) > 0 {
		for name, values := range params {
			if name == "" || values == nil || len(values) == 0 {
				continue
			}
			r.Params().Add(bytex.FromString(name), bytex.FromString(values[0]))
		}
	}

	if req.TransferEncoding != nil && len(req.TransferEncoding) > 0 {
		r.Header().Del(bytex.FromString("Transfer-Encoding"))
		for _, te := range req.TransferEncoding {
			r.Header().Add(bytex.FromString("Transfer-Encoding"), bytex.FromString(te))
		}
	}

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	b := bytex.Acquire4KBuffer()
	defer bytex.Release4KBuffer(b)
	for {
		n, readErr := req.Body.Read(b)
		if n > 0 {
			_, _ = buf.Write(b[0:n])
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			err = errors.Warning("fns: new transport request from http request failed").WithCause(readErr)
			return
		}
		if bodyLimit > 0 {
			if buf.Len() > bodyLimit {
				err = errors.Warning("fns: new transport request from http request failed").WithCause(transports.ErrTooBigRequestBody)
				return
			}
		}
	}
	r.SetBody(buf.Bytes())
	_ = req.Body.Close()
	return
}

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

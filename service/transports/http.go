/*
 * Copyright 2021 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package transports

import (
	"bufio"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
	"io"
	"net"
	"net/http"
	"strconv"
)

func HttpTransportHandlerAdaptor(h Handler, maxRequestBody int) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		r, convertErr := convertHttpRequestToRequest(request, maxRequestBody)
		if convertErr != nil {
			p, _ := json.Marshal(errors.Warning("fns: http handler adapt failed ").WithCause(convertErr))
			writer.Header().Set(contentTypeHeaderName, contentTypeJsonHeaderValue)
			writer.WriteHeader(555)
			_, _ = writer.Write(p)
			return
		}

		buf := bytebufferpool.Get()
		w := convertHttpResponseWriterToResponseWriter(writer, buf)

		h.Handle(w, r)

		for k, vv := range w.Header() {
			for _, v := range vv {
				writer.Header().Add(k, v)
			}
		}

		writer.WriteHeader(w.Status())

		if bodyLen := buf.Len(); bodyLen > 0 {
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

func convertHttpRequestToRequest(req *http.Request, bodyLimit int) (r *Request, err error) {
	r = &Request{
		ctx:        req.Context(),
		method:     bytex.FromString(req.Method),
		host:       bytex.FromString(req.Host),
		remoteAddr: bytex.FromString(req.RemoteAddr),
		header:     make(Header),
		path:       bytex.FromString(req.URL.Path),
		params:     make(RequestParams),
		body:       nil,
	}
	params := req.URL.Query()
	if params != nil && len(params) > 0 {
		for name, values := range params {
			if name == "" || values == nil || len(values) == 0 {
				continue
			}
			r.params.Add(bytex.FromString(name), bytex.FromString(values[0]))
		}
	}
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	b := acquireBuf()
	defer releaseBuf(b)
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
				err = errors.Warning("fns: new transport request from http request failed").WithCause(ErrTooBigRequestBody)
				return
			}
		}
	}
	r.body = buf.Bytes()
	_ = req.Body.Close()
	return
}

func convertHttpResponseWriterToResponseWriter(w http.ResponseWriter, buf io.Writer) ResponseWriter {
	return &netResponseWriter{
		writer:   w,
		status:   0,
		header:   make(Header),
		body:     buf,
		hijacked: false,
	}
}

type netResponseWriter struct {
	writer   http.ResponseWriter
	status   int
	header   Header
	body     io.Writer
	hijacked bool
}

func (w *netResponseWriter) Status() int {
	return w.status
}

func (w *netResponseWriter) SetStatus(status int) {
	w.status = status
}

func (w *netResponseWriter) Header() Header {
	return w.header
}

func (w *netResponseWriter) Succeed(v interface{}) {
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
		w.Header().Set(contentLengthHeaderName, strconv.Itoa(bodyLen))
		w.Header().Set(contentTypeHeaderName, contentTypeJsonHeaderValue)
		w.write(body, bodyLen)
	}
	return
}

func (w *netResponseWriter) Failed(cause errors.CodeError) {
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
		w.Header().Set(contentLengthHeaderName, strconv.Itoa(bodyLen))
		w.Header().Set(contentTypeHeaderName, contentTypeJsonHeaderValue)
		w.write(body, bodyLen)
	}
	return
}

func (w *netResponseWriter) Write(body []byte) (int, error) {
	if body == nil {
		return 0, nil
	}
	bodyLen := len(body)
	w.write(body, bodyLen)
	return bodyLen, nil
}

func (w *netResponseWriter) write(body []byte, bodyLen int) {
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

func (w *netResponseWriter) Hijack(f func(conn net.Conn, brw *bufio.ReadWriter, err error)) {
	if f == nil {
		return
	}
	if w.hijacked {
		f(nil, nil, errors.Warning("fns: connection was hijacked"))
		return
	}
	h, ok := w.writer.(http.Hijacker)
	if !ok {
		f(nil, nil, errors.Warning("fns: connection can not be hijacked"))
		return
	}
	conn, brw, err := h.Hijack()
	if err != nil {
		f(nil, nil, errors.Warning("fns: connection hijack failed").WithCause(err))
		return
	}
	w.hijacked = true
	f(conn, brw, nil)
}

func (w *netResponseWriter) Hijacked() bool {
	return w.hijacked
}

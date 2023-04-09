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
	"bytes"
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

func convertHttpRequestToRequest(req *http.Request, bodyLimit int) (r *Request, err error) {
	r = &Request{
		ctx:                req.Context(),
		isTLS:              req.URL.Scheme == httpsSchema,
		tlsConnectionState: req.TLS,
		method:             bytex.FromString(req.Method),
		host:               bytex.FromString(req.Host),
		remoteAddr:         bytex.FromString(req.RemoteAddr),
		proto:              bytex.FromString(req.Proto),
		header:             Header(req.Header),
		path:               bytex.FromString(req.URL.Path),
		params:             make(RequestParams),
		body:               nil,
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
	if req.TransferEncoding != nil && len(req.TransferEncoding) > 0 {
		delete(r.header, "Transfer-Encoding")
		for _, te := range req.TransferEncoding {
			r.header.Add("Transfer-Encoding", te)
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

func convertHttpResponseWriterToResponseWriter(w http.ResponseWriter, buf WriteBuffer) ResponseWriter {
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
	body     WriteBuffer
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

func (w *netResponseWriter) Body() []byte {
	return w.body.Bytes()
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

func (w *netResponseWriter) Hijack(f func(conn net.Conn, rw *bufio.ReadWriter) (err error)) (async bool, err error) {
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

func (w *netResponseWriter) Hijacked() bool {
	return w.hijacked
}

func ConvertRequestToHttpRequest(req *Request) (r *http.Request, err error) {
	url, urlErr := req.URL()
	if urlErr != nil {
		err = errors.Warning("fns: convert request to http request failed").WithCause(urlErr)
		return
	}
	body := req.body
	if body == nil {
		body = make([]byte, 0, 1)
	}
	r, err = http.NewRequestWithContext(req.Context(), bytex.ToString(req.method), bytex.ToString(url), bytes.NewReader(body))
	if err != nil {
		err = errors.Warning("fns: convert request to http request failed").WithCause(err)
		return
	}
	r.Proto = bytex.ToString(req.proto)
	if r.Proto == "HTTP/2" || r.Proto == "HTTP/2.0" {
		r.ProtoMajor = 2
	} else if r.Proto == "HTTP/3" || r.Proto == "HTTP/3.0" {
		r.ProtoMajor = 3
	} else {
		r.ProtoMajor = 1
	}
	r.ProtoMinor = 1

	r.Header = http.Header(req.header)
	tes, hasTE := req.header["Transfer-Encoding"]
	if hasTE {
		r.TransferEncoding = append(r.TransferEncoding, tes...)
	}

	r.TLS = req.TLSConnectionState()

	return
}

func ConvertResponseWriterToHttpResponseWriter(writer ResponseWriter) (w http.ResponseWriter) {
	w = &httpResponseWriter{
		response: writer,
	}
	return
}

type httpResponseWriter struct {
	response ResponseWriter
}

func (w *httpResponseWriter) Header() http.Header {
	return http.Header(w.response.Header())
}

func (w *httpResponseWriter) Write(bytes []byte) (int, error) {
	return w.response.Write(bytes)
}

func (w *httpResponseWriter) WriteHeader(statusCode int) {
	w.response.SetStatus(statusCode)
}

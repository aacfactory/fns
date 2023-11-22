/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package transports

import (
	"bufio"
	stdjson "encoding/json"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type ResponseWriter interface {
	context.Context
	Status() int
	SetStatus(status int)
	SetCookie(cookie *Cookie)
	Header() Header
	Succeed(v interface{})
	Failed(cause error)
	Write(body []byte) (int, error)
	Body() []byte
	BodyLen() int
	Hijack(func(ctx context.Context, conn net.Conn, rw *bufio.ReadWriter) (err error)) (async bool, err error)
	Hijacked() bool
	WriteTimeout() time.Duration
	WriteDeadline() time.Time
}

type WriteBuffer interface {
	io.Writer
	Bytes() []byte
}

var (
	responseWriterPool = sync.Pool{}
)

func AcquireResultResponseWriter(timeout time.Duration) *ResultResponseWriter {
	deadline := time.Time{}
	if timeout > 0 {
		deadline = deadline.Add(timeout)
	}
	buf := bytebufferpool.Get()
	cached := responseWriterPool.Get()
	if cached == nil {
		return &ResultResponseWriter{
			status:   0,
			timeout:  timeout,
			deadline: deadline,
			body:     buf,
		}
	}
	r := cached.(*ResultResponseWriter)
	r.body = buf
	r.timeout = timeout
	r.deadline = deadline
	return r
}

func ReleaseResultResponseWriter(w *ResultResponseWriter) {
	bytebufferpool.Put(w.body)
	w.body = nil
	w.status = 0
	w.timeout = 0
	w.deadline = time.Time{}
	responseWriterPool.Put(w)
}

type ResultResponseWriter struct {
	status   int
	timeout  time.Duration
	deadline time.Time
	body     *bytebufferpool.ByteBuffer
}

func (w *ResultResponseWriter) Status() int {
	return w.status
}

func (w *ResultResponseWriter) SetStatus(status int) {
	w.status = status
}

func (w *ResultResponseWriter) Body() []byte {
	return w.body.Bytes()
}

func (w *ResultResponseWriter) BodyLen() int {
	return w.body.Len()
}

func (w *ResultResponseWriter) Succeed(v interface{}) {
	if v == nil {
		w.status = http.StatusOK
		return
	}
	var body []byte
	var bodyErr error
	switch vv := v.(type) {
	case []byte:
		if json.Validate(vv) {
			body = vv
		} else {
			body, bodyErr = json.Marshal(v)
		}
		break
	case json.RawMessage:
		body = vv
		break
	case stdjson.RawMessage:
		body = vv
		break
	default:
		body, bodyErr = json.Marshal(v)
		break
	}
	if bodyErr != nil {
		w.Failed(errors.Warning("fns: transport write succeed result failed").WithCause(bodyErr))
		return
	}
	w.status = http.StatusOK
	if bodyLen := len(body); bodyLen > 0 {
		_, _ = w.Write(body)
	}
	return
}

func (w *ResultResponseWriter) Failed(cause error) {
	if cause == nil {
		cause = errors.Warning("fns: error is lost")
	}
	err := errors.Map(cause)
	body, encodeErr := json.Marshal(err)
	if encodeErr != nil {
		body = []byte(`{"message": "fns: transport write failed result failed"}`)
	}
	w.status = err.Code()
	if bodyLen := len(body); bodyLen > 0 {
		_, _ = w.Write(body)
	}
	return
}

func (w *ResultResponseWriter) Write(body []byte) (int, error) {
	if body == nil {
		return 0, nil
	}
	return w.body.Write(body)
}

func (w *ResultResponseWriter) WriteTimeout() time.Duration {
	return w.timeout
}

func (w *ResultResponseWriter) WriteDeadline() time.Time {
	return w.deadline
}

const (
	responseContextKey = "@fns:context:transports:response"
)

func WithResponse(ctx context.Context, w ResponseWriter) context.Context {
	ctx.SetLocalValue(bytex.FromString(responseContextKey), w)
	return ctx
}

func TryLoadResponseWriter(ctx context.Context) (ResponseWriter, bool) {
	w, ok := ctx.(ResponseWriter)
	if ok {
		return w, ok
	}
	v := ctx.LocalValue(bytex.FromString(responseContextKey))
	if v == nil {
		return nil, false
	}
	w, ok = v.(ResponseWriter)
	return w, ok
}

func LoadResponseWriter(ctx context.Context) ResponseWriter {
	w, ok := TryLoadResponseWriter(ctx)
	if ok {
		return w
	}
	return nil
}

func TryLoadResponseHeader(ctx context.Context) (Header, bool) {
	w, ok := TryLoadResponseWriter(ctx)
	if !ok {
		return nil, false
	}
	return w.Header(), ok
}

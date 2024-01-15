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

package standard

import (
	"bufio"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"net"
	"net/http"
	"time"
)

type ResponseWriter struct {
	context.Context
	writer   http.ResponseWriter
	header   transports.Header
	result   *transports.ResultResponseWriter
	hijacked bool
}

func (w *ResponseWriter) Status() int {
	return w.result.Status()
}

func (w *ResponseWriter) SetStatus(status int) {
	w.result.SetStatus(status)
}

func (w *ResponseWriter) SetCookie(cookie *transports.Cookie) {
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

func (w *ResponseWriter) Header() transports.Header {
	return w.header
}

func (w *ResponseWriter) Succeed(v interface{}) {
	w.result.Succeed(v)
	return
}

func (w *ResponseWriter) Failed(cause error) {
	w.result.Failed(cause)
	return
}

func (w *ResponseWriter) Write(body []byte) (int, error) {
	return w.result.Write(body)
}

func (w *ResponseWriter) Body() []byte {
	return w.result.Body()
}

func (w *ResponseWriter) BodyLen() int {
	return w.result.BodyLen()
}

func (w *ResponseWriter) ResetBody() {
	w.result.ResetBody()
}

func (w *ResponseWriter) Hijack(f func(ctx context.Context, conn net.Conn, rw *bufio.ReadWriter) (err error)) (async bool, err error) {
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
	err = f(w.Context, conn, brw)
	return
}

func (w *ResponseWriter) Hijacked() bool {
	return w.hijacked
}

func (w *ResponseWriter) WriteTimeout() time.Duration {
	return w.result.WriteTimeout()
}

func (w *ResponseWriter) WriteDeadline() time.Time {
	return w.result.WriteDeadline()
}

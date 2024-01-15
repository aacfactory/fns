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

package fast

import (
	"bufio"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/fasthttp"
	"net"
	"time"
)

type ResponseWriter struct {
	*Context
	result *transports.ResultResponseWriter
}

func (w *ResponseWriter) Status() int {
	return w.result.Status()
}

func (w *ResponseWriter) SetStatus(status int) {
	w.result.SetStatus(status)
}

func (w *ResponseWriter) SetCookie(cookie *transports.Cookie) {
	c := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(c)
	c.SetKeyBytes(cookie.Key())
	c.SetValueBytes(cookie.Value())
	c.SetExpire(cookie.Expire())
	c.SetMaxAge(cookie.MaxAge())
	c.SetDomainBytes(cookie.Domain())
	c.SetPathBytes(cookie.Path())
	c.SetHTTPOnly(cookie.HTTPOnly())
	c.SetSecure(cookie.Secure())
	c.SetSameSite(fasthttp.CookieSameSite(cookie.SameSite()))
	w.Context.Response.Header.SetCookie(c)
}

func (w *ResponseWriter) Header() transports.Header {
	return &ResponseHeader{
		&w.Context.Response.Header,
	}
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
	w.Context.Hijack(func(c net.Conn) {
		_ = f(w, c, nil)
	})
	async = true
	return
}

func (w *ResponseWriter) Hijacked() bool {
	return w.Context.Hijacked()
}

func (w *ResponseWriter) WriteTimeout() time.Duration {
	return w.result.WriteTimeout()
}

func (w *ResponseWriter) WriteDeadline() time.Time {
	return w.result.WriteDeadline()
}

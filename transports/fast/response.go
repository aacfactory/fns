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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/fasthttp"
	"net"
	"strconv"
	"time"
)

type responseWriter struct {
	ctx    *fasthttp.RequestCtx
	locals context.Entries
	result *transports.ResultResponseWriter
}

func (w *responseWriter) Deadline() (time.Time, bool) {
	return w.ctx.Deadline()
}

func (w *responseWriter) Done() <-chan struct{} {
	return w.ctx.Done()
}

func (w *responseWriter) Err() error {
	return w.ctx.Err()
}

func (w *responseWriter) Value(key any) any {
	return w.ctx.Value(key)
}

func (w *responseWriter) UserValue(key []byte) (val any) {
	return w.ctx.UserValueBytes(key)
}

func (w *responseWriter) ScanUserValue(key []byte, val any) (has bool, err error) {
	v := w.UserValue(key)
	if v == nil {
		return
	}
	s := scanner.New(v)
	err = s.Scan(val)
	if err != nil {
		err = errors.Warning("fns: scan context value failed").WithMeta("key", bytex.ToString(key)).WithCause(err)
		return
	}
	has = true
	return
}

func (w *responseWriter) SetUserValue(key []byte, val any) {
	w.ctx.SetUserValueBytes(key, val)
}

func (w *responseWriter) UserValues(fn func(key []byte, val any)) {
	w.ctx.VisitUserValues(fn)
}

func (w *responseWriter) LocalValue(key []byte) any {
	v := w.locals.Get(key)
	if v != nil {
		return v
	}
	return nil
}

func (w *responseWriter) ScanLocalValue(key []byte, val any) (has bool, err error) {
	v := w.LocalValue(key)
	if v == nil {
		return
	}
	s := scanner.New(v)
	err = s.Scan(val)
	if err != nil {
		err = errors.Warning("fns: scan context value failed").WithMeta("key", bytex.ToString(key)).WithCause(err)
		return
	}
	has = true
	return
}

func (w *responseWriter) SetLocalValue(key []byte, val any) {
	w.locals.Set(key, val)
}

func (w *responseWriter) Status() int {
	return w.result.Status()
}

func (w *responseWriter) SetStatus(status int) {
	w.result.SetStatus(status)
}

func (w *responseWriter) SetCookie(cookie *transports.Cookie) {
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
	w.ctx.Response.Header.SetCookie(c)
}

func (w *responseWriter) Header() transports.Header {
	return ResponseHeader{
		&w.ctx.Response.Header,
	}
}

func (w *responseWriter) Succeed(v interface{}) {
	w.result.Succeed(v)
	if bodyLen := w.result.BodyLen(); bodyLen > 0 {
		w.Header().Set(transports.ContentLengthHeaderName, bytex.FromString(strconv.Itoa(bodyLen)))
		w.Header().Set(transports.ContentTypeHeaderName, transports.ContentTypeJsonHeaderValue)
	}
	return
}

func (w *responseWriter) Failed(cause error) {
	w.result.Failed(cause)
	if bodyLen := w.result.BodyLen(); bodyLen > 0 {
		w.Header().Set(transports.ContentLengthHeaderName, bytex.FromString(strconv.Itoa(bodyLen)))
		w.Header().Set(transports.ContentTypeHeaderName, transports.ContentTypeJsonHeaderValue)
	}
	return
}

func (w *responseWriter) Write(body []byte) (int, error) {
	return w.result.Write(body)
}

func (w *responseWriter) Body() []byte {
	return w.result.Body()
}

func (w *responseWriter) BodyLen() int {
	return w.result.BodyLen()
}

func (w *responseWriter) Hijack(f func(ctx context.Context, conn net.Conn, rw *bufio.ReadWriter) (err error)) (async bool, err error) {
	if f == nil {
		err = errors.Warning("fns: hijack function is nil")
		return
	}
	w.ctx.Hijack(func(c net.Conn) {
		_ = f(w, c, nil)
	})
	async = true
	return
}

func (w *responseWriter) Hijacked() bool {
	return w.ctx.Hijacked()
}

func (w *responseWriter) WriteTimeout() time.Duration {
	return w.result.WriteTimeout()
}

func (w *responseWriter) WriteDeadline() time.Time {
	return w.result.WriteDeadline()
}

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
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/fasthttp"
	"sync"
	"time"
)

var (
	requestPool  = sync.Pool{}
	responsePool = sync.Pool{}
)

func handlerAdaptor(h transports.Handler, writeTimeout time.Duration) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var r *Request
		cr := requestPool.Get()
		if cr == nil {
			r = new(Request)
			r.locals = make(context.Entries, 0, 1)
		} else {
			r = cr.(*Request)
		}
		r.ctx = ctx

		var w *responseWriter
		cw := responsePool.Get()
		if cw == nil {
			w = new(responseWriter)
			w.locals = make(context.Entries, 0, 1)
		} else {
			w = cw.(*responseWriter)
		}
		w.ctx = ctx
		w.result = transports.AcquireResultResponseWriter(writeTimeout)

		h.Handle(w, r)

		ctx.SetStatusCode(w.Status())

		if bodyLen := w.BodyLen(); bodyLen > 0 {
			body := w.Body()
			n := 0
			for n < bodyLen {
				nn, writeErr := ctx.Write(body[n:])
				if writeErr != nil {
					break
				}
				n += nn
			}
		}

		transports.ReleaseResultResponseWriter(w.result)
		w.ctx = nil
		w.locals.Reset()
		w.result = nil
		responsePool.Put(w)

		r.ctx = nil
		r.locals.Reset()
		requestPool.Put(r)
	}
}

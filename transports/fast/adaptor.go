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
	ctxPool = sync.Pool{}
)

func handlerAdaptor(h transports.Handler, writeTimeout time.Duration) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var c *Context
		cc := ctxPool.Get()
		if cc == nil {
			cc = &Context{
				locals: make(context.Entries, 0, 1),
			}
		}
		c = cc.(*Context)
		c.RequestCtx = ctx
		r := Request{
			Context: c,
		}
		result := transports.AcquireResultResponseWriter(writeTimeout)
		w := ResponseWriter{
			Context: c,
			result:  result,
		}

		h.Handle(&w, &r)

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
		if !w.Hijacked() {
			// release result
			transports.ReleaseResultResponseWriter(result)
			// release ctx
			c.RequestCtx = nil
			c.locals.Reset()
			ctxPool.Put(c)
		}
	}
}

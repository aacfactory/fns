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
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/transports"
	"net/http"
	"sync"
	"time"
)

var (
	requestPool  = sync.Pool{}
	responsePool = sync.Pool{}
)

func HttpTransportHandlerAdaptor(h transports.Handler, maxRequestBody int, writeTimeout time.Duration) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := context.Acquire(request.Context())

		var r *Request
		cr := requestPool.Get()
		if cr == nil {
			r = new(Request)
		} else {
			r = cr.(*Request)
		}
		r.Context = ctx
		r.maxBodySize = maxRequestBody
		r.request = request

		var w *ResponseWriter
		cw := responsePool.Get()
		if cw == nil {
			w = new(ResponseWriter)
		} else {
			w = cw.(*ResponseWriter)
		}
		w.Context = ctx
		w.writer = writer
		w.header = WrapHttpHeader(writer.Header())
		w.result = transports.AcquireResultResponseWriter(writeTimeout)

		h.Handle(w, r)

		writer.WriteHeader(w.Status())

		if bodyLen := w.BodyLen(); bodyLen > 0 {
			body := w.Body()
			n := 0
			for n < bodyLen {
				nn, writeErr := writer.Write(body[n:])
				if writeErr != nil {
					break
				}
				n += nn
			}
		}

		if !w.Hijacked() {
			transports.ReleaseResultResponseWriter(w.result)
			w.Context = nil
			w.writer = nil
			w.header = nil
			w.result = nil
			w.hijacked = false
			responsePool.Put(w)

			r.Context = nil
			r.maxBodySize = 0
			r.request = nil
			requestPool.Put(r)

			context.Release(ctx)
		}

	})
}

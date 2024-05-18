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

package runtime

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/transports"
	"net/http"
	"sync"
)

var (
	ErrTooEarly    = errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: service is not ready, try later again")
	ErrUnavailable = errors.Unavailable("fns: server is closed")
)

func Middleware(rt *Runtime) transports.Middleware {
	return &middleware{
		log:     nil,
		rt:      rt,
		counter: sync.WaitGroup{},
	}
}

type middleware struct {
	log     logs.Logger
	rt      *Runtime
	counter sync.WaitGroup
}

func (middle *middleware) Name() string {
	return "runtime"
}

func (middle *middleware) Construct(options transports.MiddlewareOptions) error {
	middle.log = options.Log
	return nil
}

func (middle *middleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
		running, upped := middle.rt.Running()
		if !running {
			w.Header().Set(transports.ConnectionHeaderName, transports.CloseHeaderValue)
			w.Failed(ErrUnavailable)
			return
		}
		if !upped {
			w.Header().Set(transports.ResponseRetryAfterHeaderName, bytex.FromString("3"))
			w.Failed(ErrTooEarly)
			return
		}

		middle.counter.Add(1)
		// request Id
		requestId := r.Header().Get(transports.RequestIdHeaderName)
		if len(requestId) == 0 {
			requestId = uid.Bytes()
			r.Header().Set(transports.RequestIdHeaderName, requestId)
		}
		// set runtime into request context
		With(r, middle.rt)
		With(w, middle.rt)
		// set request and response into context
		transports.WithRequest(r, r)
		transports.WithResponse(r, w)
		// next
		next.Handle(w, r)
		// check hijacked
		if w.Hijacked() {
			middle.counter.Done()
			return
		}

		// done
		middle.counter.Done()
	})
}

func (middle *middleware) Close() (err error) {
	middle.counter.Wait()
	return
}

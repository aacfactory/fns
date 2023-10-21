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

package services

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/logs"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"net/http"
	"strconv"
	"time"
)

func newFnTask(service Service, hook func(task *fnTask)) *fnTask {
	return &fnTask{service: service,  hook: hook}
}

type fnTask struct {
	log           logs.Logger
	service       Service
	request       Request
	promise       futures.Promise
	hook          func(task *fnTask)
}

func (f *fnTask) begin(r Request, p futures.Promise) {
	f.request = r
	f.promise = p
}

func (f *fnTask) end() {
	f.request = nil
	f.promise = nil
}

// tracer 移到 endpoints 里去，在task前后
func (f *fnTask) Execute(ctx context.Context) {
	name, fn := f.request.Fn()
	fnLog := f.log.With("service", name).With("fn", fn).With("requestId", f.request.Id())

	t, hasTracer := GetTracer(ctx)
	var sp *Span = nil
	if hasTracer {
		sp = t.StartSpan(f.service.Name(), fnName)
		sp.AddTag("kind", "local")
	}
	// check cache when request is internal
	if f.request.Internal() && !f.request.Header().CacheControlDisabled() {
		etag, status, deadline, body, cached := cacheControlFetch(ctx, f.request)
		if cached && deadline.After(time.Now()) {
			if sp != nil {
				sp.AddTag("cached", "hit")
				sp.AddTag("etag", etag)
			}

			var err errors.CodeError
			if status != http.StatusOK {
				err = errors.Decode(body)
			}

			if sp != nil {
				sp.Finish()
				if err == nil {
					sp.AddTag("status", "OK")
					sp.AddTag("handled", "succeed")
				} else {
					sp.AddTag("status", err.Name())
					sp.AddTag("handled", "failed")
				}
			}

			if err == nil {
				f.promise.Succeed(body)
			} else {
				f.promise.Failed(err)
			}

			// cause internal, so do not report tracer, but report stats
			tryReportStats(ctx, serviceName, fnName, err, sp)
			if fnLog.DebugEnabled() {
				latency := time.Duration(0)
				if sp != nil {
					latency = sp.Latency()
				}
				handled := "succeed"
				if err != nil {
					handled = "failed"
				}
				fnLog.Debug().Caller().With("cache", "hit").With("latency", latency).Message(fmt.Sprintf("%s:%s was handled %s, cost %s", serviceName, fnName, handled, latency))
			}
			f.hook(f)
			return
		}
	}

	// prepare context
	buf := bytebufferpool.Get()
	_, _ = buf.Write(f.request.Hash())
	_, _ = buf.Write(bytex.FromString(f.request.Header().DeviceId()))
	barrierKey := strconv.FormatUint(xxhash.Sum64(buf.Bytes()), 10)
	bytebufferpool.Put(buf)

	if f..Components() != nil && len(f..
	Components()) > 0
	{
		ctx = withComponents(ctx, f..
		Components())
	}
	ctx = withLog(ctx, fnLog)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, f.handleTimeout)
	// handle
	v, err := f.barrier.Do(ctx, barrierKey, func() (result interface{}, err errors.CodeError) {
		result, err = f..
		Handle(ctx, fnName, f.request.Argument())
		if err != nil {
			err = err.WithMeta("service", serviceName).WithMeta("fn", fnName)
		}
		return
	})
	cancel()
	// finish span
	if sp != nil {
		sp.Finish()
		if err == nil {
			sp.AddTag("status", "OK")
			sp.AddTag("handled", "succeed")
		} else {
			sp.AddTag("status", err.Name())
			sp.AddTag("handled", "failed")
		}
	}
	// promise
	if err == nil {
		f.promise.Succeed(v)
	} else {
		f.promise.Failed(err)
	}

	// debug
	if fnLog.DebugEnabled() {
		latency := time.Duration(0)
		if sp != nil {
			latency = sp.Latency()
		}
		handled := "succeed"
		if err != nil {
			handled = "failed"
		}
		fnLog.Debug().Caller().With("latency", latency).Message(fmt.Sprintf("%s:%s was handled %s, cost %s", serviceName, fnName, handled, latency))
	}

	// try report
	tryReportTracer(ctx)
	tryReportStats(ctx, serviceName, fnName, err, sp)
	// task finish hook
	f.hook(f)
}

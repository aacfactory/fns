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

package service

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"net/http"
	"strconv"
	"time"
)

func newFnTask(svc Service, barrier Barrier, handleTimeout time.Duration, hook func(task *fnTask)) *fnTask {
	return &fnTask{svc: svc, barrier: barrier, handleTimeout: handleTimeout, hook: hook}
}

type fnTask struct {
	svc           Service
	barrier       Barrier
	request       Request
	promise       Promise
	handleTimeout time.Duration
	hook          func(task *fnTask)
}

func (f *fnTask) begin(r Request, p Promise) {
	f.request = r
	f.promise = p
}

func (f *fnTask) end() {
	f.request = nil
	f.promise = nil
}

func (f *fnTask) Execute(ctx context.Context) {
	rootLog := GetRuntime(ctx).RootLog()
	serviceName, fnName := f.request.Fn()
	fnLog := rootLog.With("service", serviceName).With("fn", fnName).With("requestId", f.request.Id())

	t, hasTracer := GetTracer(ctx)
	var sp *Span = nil
	if hasTracer {
		sp = t.StartSpan(f.svc.Name(), fnName)
		sp.AddTag("kind", "local")
	}
	// check cache when request is internal
	if f.request.Internal() && !f.request.Header().CacheControlDisabled() {
		etag, status, _, _, deadline, body, cached := CacheControlFetch(ctx, f.request)
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

	buf := bytebufferpool.Get()
	_, _ = buf.Write(f.request.Hash())
	_, _ = buf.Write(bytex.FromString(f.request.Header().DeviceId()))
	barrierKey := strconv.FormatUint(xxhash.Sum64(buf.Bytes()), 10)
	bytebufferpool.Put(buf)

	if f.svc.Components() != nil && len(f.svc.Components()) > 0 {
		ctx = withComponents(ctx, f.svc.Components())
	}

	ctx = withLog(ctx, fnLog)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, f.handleTimeout)
	v, err := f.barrier.Do(ctx, barrierKey, func() (result interface{}, err errors.CodeError) {
		result, err = f.svc.Handle(ctx, fnName, f.request.Argument())
		if err != nil {
			err = err.WithMeta("service", serviceName).WithMeta("fn", fnName)
		}
		return
	})
	cancel()

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
		f.promise.Succeed(v)
	} else {
		f.promise.Failed(err)
	}

	tryReportTracer(ctx)
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
		fnLog.Debug().Caller().With("latency", latency).Message(fmt.Sprintf("%s:%s was handled %s, cost %s", serviceName, fnName, handled, latency))
	}
	f.hook(f)
}

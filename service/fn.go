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
	"github.com/aacfactory/json"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"time"
)

func newFnTask(svc Service, barrier Barrier, request Request, result ResultWriter, handleTimeout time.Duration) *fnTask {
	return &fnTask{svc: svc, barrier: barrier, request: request, result: result, handleTimeout: handleTimeout}
}

type fnTask struct {
	svc           Service
	barrier       Barrier
	request       Request
	result        ResultWriter
	handleTimeout time.Duration
}

func (f *fnTask) Execute(ctx context.Context) {
	rootLog := getRuntime(ctx).log
	serviceName, fnName := f.request.Fn()

	t, hasTracer := GetTracer(ctx)
	var sp *Span = nil
	if hasTracer {
		sp = t.StartSpan(f.svc.Name(), fnName)
	}

	arg := f.request.Argument()
	buf := bytebufferpool.Get()
	_, _ = buf.Write([]byte(serviceName + fnName))
	if arg != nil {
		p, _ := json.Marshal(arg)
		if p != nil && len(p) > 0 {
			_, _ = buf.Write(p)
		}
	}
	_, _ = buf.Write(bytex.FromString(f.request.Header().DeviceId()))
	barrierKey := fmt.Sprintf("%d", xxhash.Sum64(buf.Bytes()))
	bytebufferpool.Put(buf)

	if f.svc.Components() != nil && len(f.svc.Components()) > 0 {
		ctx = withComponents(ctx, f.svc.Components())
	}
	fnLog := rootLog.With("service", serviceName).With("fn", fnName).With("requestId", f.request.Id())
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
	if err != nil {
		f.result.Failed(err)
	} else {
		f.result.Succeed(v)
	}
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
}

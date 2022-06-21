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
	"time"
)

func newFn(ctx context.Context, svc Service, fn string, argument Argument, result ResultWriter) *fnExecutor {
	return &fnExecutor{ctx: ctx, svc: svc, fn: fn, argument: argument, result: result}
}

type fnExecutor struct {
	ctx      context.Context
	svc      Service
	fn       string
	argument Argument
	result   ResultWriter
}

func (f *fnExecutor) Execute() {
	rootLog := getRuntime(f.ctx).log

	fnLog := rootLog.With("service", f.svc.Name()).With("fn", f.fn)
	req, hasReq := GetRequest(f.ctx)
	if hasReq {
		fnLog = fnLog.With("requestId", req.Id())
	}
	ctx := SetLog(f.ctx, fnLog)
	if f.svc.Components() != nil && len(f.svc.Components()) > 0 {
		ctx = setComponents(ctx, f.svc.Components())
	}
	t, hasTracer := GetTracer(ctx)
	var sp Span = nil
	if hasTracer {
		sp = t.StartSpan(f.svc.Name(), f.fn)
	}
	v, err := f.svc.Handle(ctx, f.fn, f.argument)
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
	tryReportStats(ctx, f.svc.Name(), f.fn, err, sp)
	if fnLog.DebugEnabled() {
		latency := time.Duration(0)
		if sp != nil {
			latency = sp.Latency()
		}
		handled := "succeed"
		if err != nil {
			handled = "failed"
		}
		fnLog.Debug().Caller().With("latency", latency).Message(fmt.Sprintf("%s:%s was handled %s, cost %s", f.svc.Name(), f.fn, handled, latency))
	}
}

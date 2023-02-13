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

func newFnTask(svc Service, request Request, result ResultWriter) *fnTask {
	return &fnTask{svc: svc, request: request, result: result}
}

type fnTask struct {
	svc     Service
	request Request
	result  ResultWriter
}

func (f *fnTask) Execute(ctx context.Context) {
	rootLog := getRuntime(ctx).log
	serviceName, fnName := f.request.Fn()
	fnLog := rootLog.With("service", serviceName).With("fn", fnName)
	req, hasReq := GetRequest(ctx)
	if hasReq {
		fnLog = fnLog.With("requestId", req.Id())
	}
	ctx = withLog(ctx, fnLog)
	if f.svc.Components() != nil && len(f.svc.Components()) > 0 {
		ctx = withComponents(ctx, f.svc.Components())
	}
	t, hasTracer := GetTracer(ctx)
	var sp Span = nil
	if hasTracer {
		sp = t.StartSpan(f.svc.Name(), fnName)
	}
	v, err := f.svc.Handle(ctx, fnName, f.request.Argument())
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

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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service/tracing"
	"github.com/aacfactory/workers"
)

type Endpoint interface {
	Request(ctx context.Context, fn string, argument Argument) (result Result)
}

type EndpointDiscovery interface {
	Get(ctx context.Context, service string) (endpoint Endpoint, has bool)
	GetExact(ctx context.Context, service string, id string) (endpoint Endpoint, has bool)
}

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
	ctx := setLog(f.ctx, rootLog.With("service", f.svc.Name()).With("fn", f.fn))
	if f.svc.Components() != nil && len(f.svc.Components()) > 0 {
		ctx = setComponents(ctx, f.svc.Components())
	}
	t, hasTracer := GetTracer(ctx)
	var span tracing.Span = nil
	if hasTracer {
		span = t.StartSpan(f.svc.Name(), f.fn)
	}
	v, err := f.svc.Handle(ctx, f.fn, f.argument)
	if span != nil {
		span.Finish()
		if err == nil {
			span.AddTag("status", "OK")
			span.AddTag("handled", "succeed")
		} else {
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
	}
	if err != nil {
		f.result.Failed(err)
	} else {
		f.result.Succeed(v)
	}
	tryReportStats(ctx, f.svc.Name(), f.fn, err, span)
}

func newEndpoint(ws workers.Workers, svc Service) *endpoint {
	return &endpoint{ws: ws, svc: svc}
}

type endpoint struct {
	ws  workers.Workers
	svc Service
}

func (e *endpoint) Request(ctx context.Context, fn string, argument Argument) (result Result) {
	fr := NewResult()
	if e.ws.Dispatch(newFn(ctx, e.svc, fn, argument, fr)) {
		fr.Failed(errors.Unavailable("fns: service is overload").WithMeta("fns", "overload"))
	}
	result = fr
	return
}

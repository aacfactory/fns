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
	"github.com/aacfactory/fns/service/tracing"
)

const (
	contextTracerKey = "_tracer_"
)

func GetTracer(ctx context.Context) (t tracing.Tracer, has bool) {
	vbv := ctx.Value(contextTracerKey)
	if vbv == nil {
		return
	}
	t, has = vbv.(tracing.Tracer)
	return
}

func setTracer(ctx context.Context) context.Context {
	r, hasRequest := GetRequest(ctx)
	if !hasRequest || r == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: get not set tracer into context cause no request found in context")))
		return ctx
	}
	ctx = context.WithValue(ctx, contextTracerKey, newTracer(r.Id()))
	return ctx
}

func newTracer(id string) (t *tracer) {

	return
}

type tracer struct {
}

func (t *tracer) Id() (id string) {
	//TODO implement me
	panic("implement me")
}

func (t *tracer) StartSpan(service string, fn string) (span tracing.Span) {
	//TODO implement me
	panic("implement me")
}

func (t *tracer) RootSpan() (span tracing.Span) {
	//TODO implement me
	panic("implement me")
}

func (t *tracer) FlatSpans() (spans tracing.Span) {
	//TODO implement me
	panic("implement me")
}

func (t *tracer) SpanSize() (size int) {
	//TODO implement me
	panic("implement me")
}

func tryReportTracer(ctx context.Context) {
	t, hasTracer := GetTracer(ctx)
	if !hasTracer {
		return
	}
	r, hasRequest := GetRequest(ctx)
	if !hasRequest {
		return
	}
	if r.Internal() {
		return
	}
	TryFork(ctx, &reportTracerTask{
		t: t,
	})
}

type reportTracerTask struct {
	t tracing.Tracer
}

func (task *reportTracerTask) Name() (name string) {
	name = "tracer-reporter"
	return
}

func (task *reportTracerTask) Execute(ctx context.Context) {
	ts, hasService := GetEndpoint(ctx, "tracings")
	if !hasService {
		return
	}
	_ = ts.Request(ctx, "report", NewArgument(task.t))
}

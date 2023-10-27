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
	"github.com/aacfactory/fns/commons/uid"
	"sync"
	"time"
)

const (
	contextTracerKey = "@fns:context:tracer"
)

func GetTracer(ctx context.Context) (t Tracer, has bool) {
	vbv := ctx.Value(contextTracerKey)
	if vbv == nil {
		return
	}
	t, has = vbv.(Tracer)
	return
}

func withTracer(ctx context.Context, id string) context.Context {
	_, has := GetTracer(ctx)
	if has {
		return ctx
	}
	ctx = context.WithValue(ctx, contextTracerKey, NewTracer(id))
	return ctx
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
	ts, hasService := GetEndpoint(ctx, "tracings")
	if !hasService {
		return
	}

	rootSpan := t.RootSpan()
	if rootSpan == nil {
		return
	}
	if rootSpan.FinishedAT().IsZero() {
		return
	}
	TryFork(ctx, &reportTracerTask{
		t:        t,
		endpoint: ts,
	})
}

type reportTracerTask struct {
	t        Tracer
	endpoint Endpoint
}

func (task *reportTracerTask) Name() (name string) {
	name = "tracer-reporter"
	return
}

func (task *reportTracerTask) Execute(ctx context.Context) {
	_ = task.endpoint.Request(ctx, NewRequest(ctx, "tracings", "report", NewArgument(task.t)))
}

type Tracer interface {
	Id() (id string)
	StartSpan(service string, fn string) (span *Span)
	Span() (span *Span)
	RootSpan() (span *Span)
}

func NewTracer(id string) Tracer {
	return &tracer{
		Id_:     id,
		Root:    nil,
		lock:    sync.Mutex{},
		current: nil,
	}
}

type tracer struct {
	Id_     string `json:"id"`
	Root    *Span  `json:"Span"`
	lock    sync.Mutex
	current *Span
}

func (t *tracer) Id() (id string) {
	id = t.Id_
	return
}

func (t *tracer) StartSpan(service string, fn string) (span *Span) {
	t.lock.Lock()
	if t.current == nil {
		t.Root = newSpan(t.Id_, service, fn, nil)
		t.current = t.Root
		t.lock.Unlock()
		return
	}
	if t.current.FinishedAT().IsZero() {
		// current not finished, current as parent
		span = newSpan(t.Id_, service, fn, t.current)
	} else {
		// current finished, current's unfinished parent as parent
		parent := t.current.Parent()
		for {
			if parent.FinishedAT().IsZero() {
				break
			}
			parent = parent.Parent()
		}
		span = newSpan(t.Id_, service, fn, parent)
	}
	t.current = span
	t.lock.Unlock()
	return
}

func (t *tracer) Span() (span *Span) {
	span = t.current
	return
}

func (t *tracer) RootSpan() (span *Span) {
	span = t.Root
	return
}

func newSpan(traceId string, service string, fn string, parent *Span) *Span {
	s := &Span{
		Id_:         uid.UID(),
		Service_:    service,
		Fn_:         fn,
		TracerId_:   traceId,
		StartAT_:    time.Now(),
		FinishedAT_: time.Time{},
		parent:      parent,
		Children_:   make([]*Span, 0, 1),
		Tags_:       make(map[string]string),
	}
	if parent != nil {
		parent.AppendChild(s)
	}
	return s
}

type Span struct {
	Id_         string            `json:"id"`
	Service_    string            `json:"service"`
	Fn_         string            `json:"fn"`
	TracerId_   string            `json:"tracerId"`
	StartAT_    time.Time         `json:"startAt"`
	FinishedAT_ time.Time         `json:"finishedAt"`
	Latency_    string            `json:"latency"`
	Children_   []*Span           `json:"children"`
	Tags_       map[string]string `json:"tags"`
	parent      *Span
}

func (s *Span) Id() (v string) {
	v = s.Id_
	return
}

func (s *Span) TracerId() (v string) {
	v = s.TracerId_
	return
}

func (s *Span) Finish() {
	s.FinishedAT_ = time.Now()
	s.Latency_ = s.FinishedAT_.Sub(s.StartAT_).String()
}

func (s *Span) AddTag(key string, value string) {
	s.Tags_[key] = value
}

func (s *Span) Tags() (v map[string]string) {
	v = s.Tags_
	return
}

func (s *Span) StartAT() (v time.Time) {
	v = s.StartAT_
	return
}

func (s *Span) FinishedAT() (v time.Time) {
	v = s.FinishedAT_
	return
}

func (s *Span) Latency() (v time.Duration) {
	if s.Latency_ != "" {
		v, _ = time.ParseDuration(s.Latency_)
	} else {
		v = s.FinishedAT_.Sub(s.StartAT_)
		s.Latency_ = v.String()
	}
	return
}

func (s *Span) Parent() (v *Span) {
	v = s.parent
	return
}

func (s *Span) Children() (v []*Span) {
	v = s.Children_
	return
}

func (s *Span) AppendChild(children ...*Span) {
	s.Children_ = append(s.Children_, children...)
	return
}

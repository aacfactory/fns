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
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/json"
	"sync"
	"time"
)

const (
	contextTracerKey = "@fns_tracer"
)

func GetTracer(ctx context.Context) (t Tracer, has bool) {
	vbv := ctx.Value(contextTracerKey)
	if vbv == nil {
		return
	}
	t, has = vbv.(Tracer)
	return
}

func withTracer(ctx context.Context) context.Context {
	_, has := GetTracer(ctx)
	if has {
		return ctx
	}
	r, hasRequest := GetRequest(ctx)
	if !hasRequest || r == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: get not set tracer into context cause no request found in context")))
		return ctx
	}
	ctx = context.WithValue(ctx, contextTracerKey, NewTracer(r.Id()))
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
	StartSpan(service string, fn string) (span Span)
	Span() (span Span)
	RootSpan() (span Span)
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
	Root    Span   `json:"span"`
	lock    sync.Mutex
	current Span
}

func (t *tracer) Id() (id string) {
	id = t.Id_
	return
}

func (t *tracer) StartSpan(service string, fn string) (span Span) {
	t.lock.Lock()
	if t.current == nil {
		span = newSpan(t.Id_, service, fn, nil)
		t.Root = span
		t.current = span
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

func (t *tracer) Span() (span Span) {
	span = t.current
	return
}

func (t *tracer) RootSpan() (span Span) {
	span = t.Root
	return
}

type Span interface {
	Id() (v string)
	TracerId() (v string)
	Finish()
	AddTag(key string, value string)
	Tags() (v map[string]string)
	StartAT() (v time.Time)
	FinishedAT() (v time.Time)
	Latency() (v time.Duration)
	Parent() (v Span)
	Children() (v []Span)
	AppendChild(children ...Span)
}

func DecodeSpan(p []byte) (v Span, err errors.CodeError) {
	s := &span{}
	decodeErr := json.Unmarshal(p, s)
	if decodeErr != nil {
		err = errors.Warning("fns: decode span failed").WithCause(decodeErr)
		return
	}
	v = s
	return
}

func newSpan(traceId string, service string, fn string, parent Span) Span {
	s := &span{
		Id_:         uid.UID(),
		Service_:    service,
		Fn_:         fn,
		TracerId_:   traceId,
		StartAT_:    time.Now(),
		FinishedAT_: time.Time{},
		parent:      parent,
		Children_:   make([]Span, 0, 1),
		Tags_:       make(map[string]string),
	}
	if parent != nil {
		parent.AppendChild(s)
	}
	return s
}

type span struct {
	Id_         string    `json:"id"`
	Service_    string    `json:"service"`
	Fn_         string    `json:"fn"`
	TracerId_   string    `json:"tracerId"`
	StartAT_    time.Time `json:"startAt"`
	FinishedAT_ time.Time `json:"finishedAt"`
	parent      Span
	Children_   []Span            `json:"children"`
	Tags_       map[string]string `json:"tags"`
}

func (s *span) Id() (v string) {
	v = s.Id_
	return
}

func (s *span) TracerId() (v string) {
	v = s.TracerId_
	return
}

func (s *span) Finish() {
	s.FinishedAT_ = time.Now()
}

func (s *span) AddTag(key string, value string) {
	s.Tags_[key] = value
}

func (s *span) Tags() (v map[string]string) {
	v = s.Tags_
	return
}

func (s *span) StartAT() (v time.Time) {
	v = s.StartAT_
	return
}

func (s *span) FinishedAT() (v time.Time) {
	v = s.FinishedAT_
	return
}

func (s *span) Latency() (v time.Duration) {
	v = s.FinishedAT_.Sub(s.StartAT_)
	return
}

func (s *span) Parent() (v Span) {
	v = s.parent
	return
}

func (s *span) Children() (v []Span) {
	v = s.Children_
	return
}

func (s *span) AppendChild(children ...Span) {
	s.Children_ = append(s.Children_, children...)
	return
}

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

package tracing

import "sync"

type Tracer interface {
	Id() (id string)
	StartSpan(service string, fn string) (span Span)
	Span() (span Span)
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
	if t.Root == nil {
		span = newSpan(t.Id_, service, fn, nil)
		t.Root = span
		t.current = span
		t.lock.Unlock()
		return
	}
	span = newSpan(t.Id_, service, fn, t.current)
	if t.current.FinishedAT().IsZero() {
		// prev not finished, current as parent
		span = newSpan(t.Id_, service, fn, t.current)
	} else {
		// pre finished, current's parent as parent
		span = newSpan(t.Id_, service, fn, t.current.Parent())
		// new as current
		t.current = span
	}
	t.lock.Unlock()
	return
}

func (t *tracer) Span() (span Span) {
	span = t.Root
	return
}

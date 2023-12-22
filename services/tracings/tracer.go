/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package tracings

import (
	"github.com/aacfactory/fns/commons/bytex"
	"time"
)

type Trace struct {
	Id   string
	Span *Span
}

func New(id []byte) *Tracer {
	return &Tracer{
		Id:      bytex.ToString(id),
		Span:    nil,
		current: nil,
	}
}

type Tracer struct {
	Id      string
	Span    *Span
	current *Span
}

func (trace *Tracer) Trace() (v *Trace) {
	v = &Trace{
		Id:   trace.Id,
		Span: trace.Span,
	}
	return
}

func (trace *Tracer) Begin(pid []byte, endpoint []byte, fn []byte, tags ...string) {
	if trace.current != nil && trace.current.Id == bytex.ToString(pid) {
		return
	}
	current := &Span{
		Id:       bytex.ToString(pid),
		Endpoint: bytex.ToString(endpoint),
		Fn:       bytex.ToString(fn),
		Begin:    time.Now(),
		Waited:   time.Time{},
		End:      time.Time{},
		Tags:     make(map[string]string),
		Children: nil,
		parent:   nil,
	}
	current.setTags(tags)
	if trace.Span == nil {
		trace.Span = current
		trace.current = current
		return
	}
	parent := trace.current
	if parent.Children == nil {
		parent.Children = make([]*Span, 0, 1)
	}
	parent.Children = append(parent.Children, current)
	current.parent = parent
	trace.current = current
}

func (trace *Tracer) Waited(tags ...string) {
	if trace.current == nil {
		return
	}
	trace.current.Waited = time.Now()
	trace.current.setTags(tags)
	return
}

func (trace *Tracer) Tagging(tags ...string) {
	if trace.current == nil {
		return
	}
	trace.current.setTags(tags)
	return
}

func (trace *Tracer) Finish(tags ...string) {
	if trace.current == nil {
		return
	}
	if trace.current.Waited.IsZero() {
		trace.current.Waited = trace.current.Begin
	}
	trace.current.End = time.Now()
	trace.current.setTags(tags)
	if trace.current.parent != nil {
		trace.current = trace.current.parent
	}
}

func (trace *Tracer) Mount(child *Span) {
	if trace.current == nil {
		return
	}
	if child == nil {
		return
	}
	if trace.current.Children == nil {
		trace.current.Children = make([]*Span, 0, 1)
	}
	child.parent = trace.current
	child.mountChildrenParent()
	trace.current.Children = append(trace.current.Children, child)
}

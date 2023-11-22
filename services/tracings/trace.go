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

func New(id []byte) Trace {
	return Trace{
		Id:      bytex.ToString(id),
		Span:    nil,
		current: nil,
	}
}

type Trace struct {
	Id      string `json:"id"`
	Span    *Span  `json:"span"`
	current *Span
}

func (trace *Trace) Begin(pid []byte, endpoint []byte, fn []byte, tags ...string) {
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

func (trace *Trace) Waited(tags ...string) {
	if trace.current == nil {
		return
	}
	trace.current.Waited = time.Now()
	trace.current.setTags(tags)
	return
}

func (trace *Trace) Tagging(tags ...string) {
	if trace.current == nil {
		return
	}
	trace.current.setTags(tags)
	return
}

func (trace *Trace) Finish(tags ...string) {
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

func (trace *Trace) Mount(child *Span) {
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

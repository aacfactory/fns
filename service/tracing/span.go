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

import (
	"github.com/aacfactory/fns/uid"
	"time"
)

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

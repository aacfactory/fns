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

package tracings

import (
	"time"
)

type Tracer struct {
	Id   string `json:"id" validate:"not_blank" message:"id is blank"`
	Span *Span  `json:"span" validate:"required" message:"span is required"`
}

func (tracer *Tracer) FlatSpans() (spans []*Span) {
	if tracer.Span == nil {
		return
	}
	spans = tracer.flatSpans(tracer.Span)
	return
}

func (tracer *Tracer) flatSpans(span *Span) (spans []*Span) {
	spans = make([]*Span, 0, 1)
	cp := &Span{
		Id:         span.Id,
		Service:    span.Service,
		Fn:         span.Fn,
		TracerId:   span.TracerId,
		StartAT:    span.StartAT,
		FinishedAT: span.FinishedAT,
		Parent:     nil,
		Children:   nil,
		Tags:       span.Tags,
	}
	spans = append(spans, cp)
	if span.Children != nil {
		for _, child := range span.Children {
			children := tracer.flatSpans(child)
			for _, s := range children {
				s.Parent = cp
			}
			spans = append(spans, children...)
		}
	}
	return
}

type Span struct {
	Id         string            `json:"id" validate:"not_blank" message:"id is blank"`
	Service    string            `json:"service" validate:"not_blank" message:"service is blank"`
	Fn         string            `json:"fn" validate:"not_blank" message:"fn is blank"`
	TracerId   string            `json:"tracerId" validate:"not_blank" message:"tracerId is blank"`
	StartAT    time.Time         `json:"startAt" validate:"required" message:"startAt is invalid"`
	FinishedAT time.Time         `json:"finishedAt" validate:"required" message:"finishedAt is invalid"`
	Parent     *Span             `json:"parent"`
	Children   []*Span           `json:"children"`
	Tags       map[string]string `json:"tags"`
}

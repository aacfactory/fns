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

package fns

import "time"

type Span interface {
	Id() (v string)
	TracerId() (v string)
	Finish()
	AddTag(key string, value interface{})
	Tags() (v map[string]interface{})
	StartAT() (v time.Time)
	FinishedAT() (v time.Time)
	Latency() (v time.Duration)
	Parent() (v Span)
	Children() (v []Span)
	AppendChild(children ...Span)
}

type Tracer interface {
	Id() (id string)
	StartSpan(service string, fn string) (span Span)
	RootSpan() (span Span)
	FlatSpans() (spans Span)
	SpanSize() (size int)
}

func newTracer(id string) (v Tracer) {

	return
}

type TracerReporter interface {
	Build(env Environments) (err error)
	Report(tracer Tracer)
	Close() (err error)
}

func defaultTracerReporter() (reporter TracerReporter) {
	return
}

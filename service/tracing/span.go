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

import "time"

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

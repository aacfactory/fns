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

import "time"

type Span struct {
	Id       string            `json:"id"`
	Endpoint string            `json:"endpoint"`
	Fn       string            `json:"fn"`
	Begin    time.Time         `json:"begin"`
	Waited   time.Time         `json:"waited"`
	End      time.Time         `json:"end"`
	Tags     map[string]string `json:"tags"`
	Children []*Span           `json:"children"`
	parent   *Span
}

func (span *Span) setTags(tags []string) {
	n := len(tags)
	if n == 0 {
		return
	}
	if n%2 != 0 {
		return
	}
	for i := 0; i < n; i += 2 {
		k := tags[i]
		v := tags[i+1]
		span.Tags[k] = v
	}
}

func (span *Span) mountChildrenParent() {
	for _, child := range span.Children {
		if child.parent == nil {
			child.parent = span
		}
		child.mountChildrenParent()
	}
}

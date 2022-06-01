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

package documents

func NewFn(name string, title string, description string, hasAuthorization bool, deprecated bool) *Fn {
	return &Fn{
		Name:             name,
		Title:            title,
		Description:      description,
		HasAuthorization: hasAuthorization,
		Argument:         nil,
		Result:           nil,
		Deprecated:       deprecated,
	}
}

type Fn struct {
	Name             string   `json:"name,omitempty"`
	Title            string   `json:"title,omitempty"`
	Description      string   `json:"description,omitempty"`
	HasAuthorization bool     `json:"hasAuthorization,omitempty"`
	Argument         *Element `json:"argument,omitempty"`
	Result           *Element `json:"result,omitempty"`
	Deprecated       bool     `json:"deprecated,omitempty"`
}

func (doc *Fn) SetArgument(v *Element) {
	doc.Argument = v
}

func (doc *Fn) SetResult(v *Element) {
	doc.Result = v
}

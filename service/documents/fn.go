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

func newFn(name string, title string, description string, authorization bool, deprecated bool, arg *Element, result *Element) *Fn {
	return &Fn{
		Name_:          name,
		Title_:         title,
		Description_:   description,
		Authorization_: authorization,
		Argument_:      arg,
		Result_:        result,
		Deprecated_:    deprecated,
	}
}

type Fn struct {
	Name_          string   `json:"name,omitempty"`
	Title_         string   `json:"title,omitempty"`
	Description_   string   `json:"description,omitempty"`
	Authorization_ bool     `json:"authorization,omitempty"`
	Argument_      *Element `json:"argument,omitempty"`
	Result_        *Element `json:"result,omitempty"`
	Deprecated_    bool     `json:"deprecated,omitempty"`
}

func (fn *Fn) Name() (name string) {
	name = fn.Name_
	return
}

func (fn *Fn) Title() (title string) {
	title = fn.Title_
	return
}

func (fn *Fn) Description() (description string) {
	description = fn.Description_
	return
}

func (fn *Fn) Authorization() (has bool) {
	has = fn.Authorization_
	return
}

func (fn *Fn) Deprecated() (deprecated bool) {
	deprecated = fn.Deprecated_
	return
}

func (fn *Fn) Argument() (argument *Element) {
	argument = fn.Argument_
	return
}

func (fn *Fn) Result() (result *Element) {
	result = fn.Result_
	return
}

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

import (
	"net/http"
	"sort"
	"strings"
)

func NewFn(name string) Fn {
	return Fn{
		Name:          name,
		Title:         "",
		Description:   "",
		Deprecated:    false,
		Methods:       []string{http.MethodPost},
		Authorization: false,
		Param:         Nil(),
		Result:        Nil(),
		Errors:        make(Errors, 0, 1),
	}
}

type Fn struct {
	Name          string   `json:"name,omitempty"`
	Title         string   `json:"title,omitempty"`
	Description   string   `json:"description,omitempty"`
	Deprecated    bool     `json:"deprecated,omitempty"`
	Methods       []string `json:"methods,omitempty"`
	Authorization bool     `json:"authorization,omitempty"`
	Param         Element  `json:"argument,omitempty"`
	Result        Element  `json:"result,omitempty"`
	Errors        Errors   `json:"errors,omitempty"`
}

func (fn Fn) SetInfo(title string, description string) Fn {
	fn.Title = title
	fn.Description = description
	return fn
}

func (fn Fn) SetMethod(method ...string) Fn {
	fn.Methods = method
	return fn
}

func (fn Fn) SetAuthorization() Fn {
	fn.Authorization = true
	return fn
}

func (fn Fn) SetDeprecated() Fn {
	fn.Deprecated = true
	return fn
}

func (fn Fn) SetParam(param Element) Fn {
	fn.Param = param
	return fn
}

func (fn Fn) SetResult(result Element) Fn {
	fn.Result = result
	return fn
}

func (fn Fn) AddError(err Error) Fn {
	fn.Errors = fn.Errors.Add(err)
	return fn
}

type Fns []Fn

func (pp Fns) Len() int {
	return len(pp)
}

func (pp Fns) Less(i, j int) bool {
	return strings.Compare(pp[i].Name, pp[j].Name) < 0
}

func (pp Fns) Swap(i, j int) {
	pp[i], pp[j] = pp[j], pp[i]
}

func (pp Fns) Add(fn Fn) Fns {
	n := append(pp, fn)
	sort.Sort(n)
	return n
}

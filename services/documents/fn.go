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

package documents

import (
	"sort"
	"strings"
)

func NewFn(name string) Fn {
	return Fn{
		Name:          name,
		Title:         "",
		Description:   "",
		Deprecated:    false,
		Readonly:      false,
		Authorization: false,
		Param:         Unknown(),
		Result:        Unknown(),
		Errors:        make(Errors, 0, 1),
	}
}

type Fn struct {
	Name          string  `json:"name,omitempty"`
	Title         string  `json:"title,omitempty"`
	Description   string  `json:"description,omitempty"`
	Deprecated    bool    `json:"deprecated,omitempty"`
	Readonly      bool    `json:"readonly,omitempty"`
	Internal      bool    `json:"internal,omitempty"`
	Authorization bool    `json:"authorization,omitempty"`
	Permission    bool    `json:"permission,omitempty"`
	Param         Element `json:"argument,omitempty"`
	Result        Element `json:"result,omitempty"`
	Errors        Errors  `json:"errors,omitempty"`
}

func (fn Fn) SetInfo(title string, description string) Fn {
	fn.Title = title
	fn.Description = description
	return fn
}

func (fn Fn) SetDeprecated(deprecated bool) Fn {
	fn.Deprecated = deprecated
	return fn
}

func (fn Fn) SetReadonly(readonly bool) Fn {
	fn.Readonly = readonly
	return fn
}

func (fn Fn) SetInternal(internal bool) Fn {
	fn.Internal = internal
	return fn
}

func (fn Fn) SetAuthorization(authorization bool) Fn {
	fn.Authorization = authorization
	return fn
}

func (fn Fn) SetPermission(permission bool) Fn {
	fn.Permission = permission
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

func (fn Fn) SetErrors(err string) Fn {
	fn.Errors = NewErrors(err)
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

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

func newFn(name string, title string, description string, authorization bool, deprecated bool, arg *Element, result *Element, errs []Error) *Fn {
	return &Fn{
		Name:          name,
		Title:         title,
		Description:   description,
		Authorization: authorization,
		Argument:      arg,
		Result:        result,
		Deprecated:    deprecated,
		Errors:        errs,
	}
}

type Fn struct {
	Name          string   `json:"name,omitempty"`
	Title         string   `json:"title,omitempty"`
	Description   string   `json:"description,omitempty"`
	Authorization bool     `json:"authorization,omitempty"`
	Argument      *Element `json:"argument,omitempty"`
	Result        *Element `json:"result,omitempty"`
	Deprecated    bool     `json:"deprecated,omitempty"`
	Errors        []Error  `json:"errors,omitempty"`
}

type Error struct {
	Name_         string
	Descriptions_ map[string]string
}

func (e Error) Name() string {
	return e.Name_
}

func (e Error) Descriptions() map[string]string {
	return e.Descriptions_
}

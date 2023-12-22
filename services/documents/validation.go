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

type I18n struct {
	Name  string `json:"name,omitempty" avro:"name"`
	Value string `json:"value,omitempty" avro:"value"`
}

type I18ns []I18n

func (pp I18ns) Len() int {
	return len(pp)
}

func (pp I18ns) Less(i, j int) bool {
	return strings.Compare(pp[i].Name, pp[j].Name) < 0
}

func (pp I18ns) Swap(i, j int) {
	pp[i], pp[j] = pp[j], pp[i]
}

func (pp I18ns) Add(name string, value string) I18ns {
	n := append(pp, I18n{
		Name:  name,
		Value: value,
	})
	sort.Sort(n)
	return n
}

func (pp I18ns) Get(name string) (p string, has bool) {
	for _, i18n := range pp {
		if i18n.Name == name {
			p = i18n.Value
			has = true
			return
		}
	}
	return
}

func NewValidation(name string) Validation {
	return Validation{
		Enable: true,
		Name:   name,
		I18ns:  make(I18ns, 0, 1),
	}
}

type Validation struct {
	Enable bool   `json:"enable,omitempty" avro:"enable"`
	Name   string `json:"name,omitempty" avro:"name"`
	I18ns  I18ns  `json:"i18ns,omitempty" avro:"i18Ns"`
}

func (validation Validation) AddI18n(name string, value string) Validation {
	validation.I18ns = validation.I18ns.Add(name, value)
	return validation
}

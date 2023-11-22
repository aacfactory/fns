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

type Property struct {
	Name    string
	Element Element
}

type Properties []Property

func (pp Properties) Len() int {
	return len(pp)
}

func (pp Properties) Less(i, j int) bool {
	return strings.Compare(pp[i].Name, pp[j].Name) < 0
}

func (pp Properties) Swap(i, j int) {
	pp[i], pp[j] = pp[j], pp[i]
}

func (pp Properties) Add(name string, element Element) Properties {
	n := append(pp, Property{
		Name:    name,
		Element: element,
	})
	sort.Sort(n)
	return n
}

func (pp Properties) Get(name string) (p Property, has bool) {
	for _, property := range pp {
		if property.Name == name {
			p = property
			has = true
			return
		}
	}
	return
}

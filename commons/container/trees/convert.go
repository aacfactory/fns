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

package trees

import (
	"github.com/aacfactory/errors"
	"sort"
)

const (
	treeTag = "tree"
)

func ConvertListToTree[T any](items []T) (v []T, err error) {
	itemLen := len(items)
	if itemLen == 0 {
		return
	}
	elements := make(Elements[T], 0, itemLen)
	for _, item := range items {
		element, elementErr := NewElement[T](item)
		if elementErr != nil {
			err = errors.Warning("fns: convert list to tree failed").WithCause(elementErr)
			return
		}
		elements = append(elements, element)
	}
	for _, element := range elements {
		convert[T](element, elements)
	}
	sort.Sort(elements)
	for _, element := range elements {
		if element.hasParent {
			continue
		}
		v = append(v, element.Interface())
	}
	return
}

func convert[T any](element *Element[T], elements Elements[T]) {
	n := 0
	for i, other := range elements {
		if element.ident().Equal(other.parent()) {
			convert[T](other, elements)
			if !element.contains(other) {
				element.children = append(element.children, other)
			}
			other.hasParent = true
			elements[i] = other
			n++
		}
	}
	if n > 0 {
		sort.Sort(element.children)
	}
}

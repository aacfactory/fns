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
	"github.com/aacfactory/fns/commons/versions"
	"sort"
)

// NewDocument todo change with fnc(forg was changed)
func NewDocument(name string, description string, ver versions.Version) *Document {
	return &Document{
		Name:        name,
		Description: description,
		Version:     ver,
		Fns:         make([]*Fn, 0, 1),
		Elements:    make(map[string]*Element),
	}
}

type Document struct {
	// Name
	// as tag
	Name string `json:"name"`
	// Description
	// as description of tag, support markdown
	Description string `json:"description"`
	// Version
	Version versions.Version `json:"version"`
	// Fns
	Fns []*Fn `json:"fns"`
	// Elements
	Elements map[string]*Element `json:"elements"`
}

func (doc *Document) AddFn(name string, title string, description string, hasAuthorization bool, deprecated bool, arg *Element, result *Element, errs []Error) {
	argRef := doc.addElement(arg)
	resultRef := doc.addElement(result)
	doc.Fns = append(doc.Fns, newFn(name, title, description, hasAuthorization, deprecated, argRef, resultRef, errs))
}

func (doc *Document) addElement(element *Element) (ref *Element) {
	if element == nil {
		return
	}
	unpacks := element.unpack()
	ref = unpacks[0]
	if len(unpacks) <= 1 {
		return
	}
	remains := unpacks[1:]
	sort.Sort(remains)
	for _, remain := range remains {
		if remain.isBuiltin() || remain.isRef() || remain.Path == "" {
			continue
		}
		key := remain.Key()
		if _, has := doc.Elements[key]; !has {
			doc.Elements[key] = remain
		}
	}
	return
}

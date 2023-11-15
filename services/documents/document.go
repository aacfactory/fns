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
)

func New(name string, description string, internal bool, ver versions.Version) Document {
	return Document{
		Name:        name,
		Description: description,
		Internal:    internal,
		Version:     ver,
		Functions:   make(Fns, 0, 1),
		Elements:    make(Elements, 0, 1),
	}
}

type Document struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Internal    bool             `json:"internal"`
	Version     versions.Version `json:"version"`
	Functions   Fns              `json:"functions"`
	Elements    Elements         `json:"elements"`
}

func (doc *Document) IsEmpty() bool {
	return doc.Name == ""
}

func (doc *Document) AddFn(fn Fn) {
	if fn.Param.Exist() {
		paramRef := doc.addElement(fn.Param)
		fn.Param = paramRef
	}
	if fn.Result.Exist() {
		paramRef := doc.addElement(fn.Result)
		fn.Result = paramRef
	}
	if doc.Internal {
		fn.Internal = true
	}
	doc.Functions = doc.Functions.Add(fn)
}

func (doc *Document) addElement(element Element) (ref Element) {
	if !element.Exist() {
		return
	}
	unpacks := element.unpack()
	ref = unpacks[0]
	if len(unpacks) <= 1 {
		return
	}
	remains := unpacks[1:]
	for _, remain := range remains {
		if remain.isBuiltin() || remain.isRef() || remain.Path == "" {
			continue
		}
		doc.Elements = doc.Elements.Add(remain)
	}
	return
}

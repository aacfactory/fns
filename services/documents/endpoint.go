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

import "github.com/aacfactory/fns/commons/versions"

func New(name string, title string, description string) Endpoint {
	return Endpoint{
		Name:        name,
		Title:       title,
		Description: description,
		Functions:   make(Fns, 0, 1),
		Elements:    make(Elements, 0, 1),
	}
}

type Endpoint struct {
	Id          []string         `json:"id,omitempty"`
	Version     versions.Version `json:"version"`
	Name        string           `json:"name"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Internal    bool             `json:"internal"`
	Functions   Fns              `json:"functions"`
	Elements    Elements         `json:"elements"`
}

func (endpoint *Endpoint) Defined() bool {
	return endpoint.Name != ""
}

func (endpoint *Endpoint) SetInternal() {
	endpoint.Internal = true
}

func (endpoint *Endpoint) AddFn(fn Fn) {
	if fn.Param.Exist() {
		if !fn.Readonly {
			paramRef := endpoint.addElement(fn.Param)
			fn.Param = paramRef
		}
	}
	if fn.Result.Exist() {
		paramRef := endpoint.addElement(fn.Result)
		fn.Result = paramRef
	}
	endpoint.Functions = endpoint.Functions.Add(fn)
}

func (endpoint *Endpoint) addElement(element Element) (ref Element) {
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
		if remain.IsBuiltin() || remain.IsRef() || remain.Path == "" {
			continue
		}
		endpoint.Elements = endpoint.Elements.Add(remain)
	}
	return
}

type Endpoints []Endpoint

func (endpoints Endpoints) Len() int {
	return len(endpoints)
}

func (endpoints Endpoints) Less(i, j int) bool {
	return endpoints[i].Name < endpoints[j].Name
}

func (endpoints Endpoints) Swap(i, j int) {
	endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
	return
}

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
	"github.com/aacfactory/fns/commons/versions"
)

func New(name string, title string, description string, version versions.Version) Endpoint {
	return Endpoint{
		Version:     version,
		Name:        name,
		Title:       title,
		Description: description,
		Functions:   make(Fns, 0, 1),
		Elements:    make(Elements, 0, 1),
	}
}

type Endpoint struct {
	Version     versions.Version `json:"version" avro:"version"`
	Name        string           `json:"name" avro:"name"`
	Title       string           `json:"title" avro:"title"`
	Description string           `json:"description" avro:"description"`
	Internal    bool             `json:"internal" avro:"internal"`
	Functions   Fns              `json:"functions" avro:"functions"`
	Elements    Elements         `json:"elements" avro:"elements"`
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
		ref = element
		return
	}
	if element.IsRef() || element.IsAny() {
		ref = element
		return
	}
	unpacks := unpack(element)
	ref = unpacks[0]
	for _, unpacked := range unpacks {
		if unpacked.IsBuiltin() || unpacked.IsRef() || unpacked.Path == "" {
			continue
		}
		endpoint.Elements = endpoint.Elements.Add(unpacked)
	}
	return
}

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
	"github.com/aacfactory/fns/service"
	"sort"
)

func NewService(name string, description string) *Service {
	return &Service{
		Name_:        name,
		Description_: description,
		Fns_:         make([]service.FnDocument, 0, 1),
		Elements_:    make(map[string]service.ElementDocument),
	}
}

type Service struct {
	// Name
	// as tag
	Name_ string `json:"name"`
	// Description
	// as description of tag, support markdown
	Description_ string `json:"description"`
	// Fns
	Fns_ []service.FnDocument `json:"fns"`
	// Elements
	Elements_ map[string]service.ElementDocument `json:"elements"`
}

func (svc *Service) Name() (name string) {
	name = svc.Name_
	return
}

func (svc *Service) Description() (description string) {
	description = svc.Description_
	return
}

func (svc *Service) Fns() (fns []service.FnDocument) {
	fns = svc.Fns_
	return
}

func (svc *Service) Elements() (elements map[string]service.ElementDocument) {
	elements = svc.Elements_
	return
}

func (svc *Service) AddFn(name string, title string, description string, hasAuthorization bool, deprecated bool, arg *Element, result *Element, errs []FnError) {
	argRef := svc.addElement(arg)
	resultRef := svc.addElement(result)
	svc.Fns_ = append(svc.Fns_, newFn(name, title, description, hasAuthorization, deprecated, argRef, resultRef, errs))
}

func (svc *Service) addElement(element *Element) (ref *Element) {
	if element == nil {
		return
	}
	ref = element.mapToRef()
	objects := element.objects()
	if objects == nil || len(objects) == 0 {
		return
	}
	keys := make([]string, 0, 1)
	for key := range objects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if obj, has := svc.Elements_[key]; !has {
			svc.Elements_[key] = obj
		}
	}
	return
}

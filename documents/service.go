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

func NewService(name string, description string) *Service {
	return &Service{
		Name:        name,
		Description: description,
		Fns:         make(map[string]*Fn),
	}
}

type Service struct {
	// Name
	// as tag
	Name string `json:"name"`
	// Description
	// as description of tag, support markdown
	Description string `json:"description"`
	// Fns
	// Key: fn
	Fns map[string]*Fn `json:"fns"`
}

func (doc *Service) AddFn(fn *Fn) {
	doc.Fns[fn.Name] = fn
}

func (doc *Service) Elements() (v map[string]*Element) {
	v = make(map[string]*Element)
	if doc.Fns == nil || len(doc.Fns) == 0 {
		return
	}

	for _, fn := range doc.Fns {
		// argument
		argObjects := fn.Argument.objects()
		if argObjects != nil && len(argObjects) > 0 {
			for k, obj := range argObjects {
				if _, has := v[k]; !has {
					v[k] = obj
				}
			}
		}
		// result
		resultObjects := fn.Result.objects()
		if resultObjects != nil && len(resultObjects) > 0 {
			for k, obj := range resultObjects {
				if _, has := v[k]; !has {
					v[k] = obj
				}
			}
		}
	}
	return
}

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

package rbac

import (
	"github.com/aacfactory/fns/commons/wildcard"
)

type Policy struct {
	Object  string `json:"object"`
	Action  string `json:"action"`
	matcher *wildcard.Wildcard
}

func (p *Policy) match(action string) (ok bool) {
	if p.matcher == nil {
		ok = true
		return
	}
	ok = p.matcher.Match(action)
	return
}

type Role struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Parent      string    `json:"parent"`
	Children    []*Role   `json:"children"`
	Policies    []*Policy `json:"policies"`
}

func (r *Role) enforce(object string, action string) (ok bool) {
	if r.Policies != nil && len(r.Policies) > 0 {
		for _, policy := range r.Policies {
			if policy.Object != object {
				continue
			}
			ok = policy.match(action)
			if ok {
				return
			}
		}
	}
	if r.Children != nil && len(r.Children) > 0 {
		for _, child := range r.Children {
			ok = child.enforce(object, action)
			if ok {
				return
			}
		}
	}
	return
}

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

package cluster

import "github.com/aacfactory/json"

type ResourceUpdateEvent struct {
	Node  string          `json:"node"`
	Kind  string          `json:"kind"`
	Value json.RawMessage `json:"value"`
}

type ResourcesHandler interface {
	Handle(e *ResourceUpdateEvent)
}

type Resources interface {
	Updated(kind string, value interface{}) (err error)
	RegisterHandler(kind string, handler ResourcesHandler) (err error)
}

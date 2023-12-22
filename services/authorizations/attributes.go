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

package authorizations

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
)

type Attribute struct {
	Key   []byte          `json:"key" avro:"key"`
	Value json.RawMessage `json:"value" avro:"value"`
}

type Attributes []Attribute

func (attributes *Attributes) Get(key []byte, value interface{}) (has bool, err error) {
	attrs := *attributes
	for _, attribute := range attrs {
		if bytes.Equal(key, attribute.Key) {
			decodeErr := json.Unmarshal(attribute.Value, value)
			if decodeErr != nil {
				err = errors.Warning("authorizations: attributes get failed").WithCause(decodeErr).WithMeta("key", string(key))
				return
			}
			has = true
			return
		}
	}
	return
}

func (attributes *Attributes) Set(key []byte, value interface{}) (err error) {
	p, encodeErr := json.Marshal(value)
	if encodeErr != nil {
		err = errors.Warning("authorizations: attributes set failed").WithCause(encodeErr).WithMeta("key", string(key))
		return
	}
	attrs := *attributes
	attrs = append(attrs, Attribute{
		Key:   key,
		Value: p,
	})
	*attributes = attrs
	return
}

func (attributes *Attributes) Remove(key []byte) {
	attrs := *attributes
	n := -1
	for i, attribute := range attrs {
		if bytes.Equal(key, attribute.Key) {
			n = i
			break
		}
	}
	if n == -1 {
		return
	}
	attrs = append(attrs[:n], attrs[n+1:]...)
	*attributes = attrs
	return
}

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

package json

import (
	jsoniter "github.com/json-iterator/go"
)

var (
	_json jsoniter.API
)

func init() {
	_json = jsoniter.ConfigCompatibleWithStandardLibrary
}

type Marshaler interface {
	MarshalJSON() ([]byte, error)
}

type Unmarshaler interface {
	UnmarshalJSON([]byte) error
}

func API() jsoniter.API {
	return _json
}

func Validate(data []byte) bool {
	return jsoniter.Valid(data)
}

func ValidateString(data string) bool {
	return jsoniter.Valid([]byte(data))
}

func Marshal(v interface{}) (p []byte, err error) {
	p, err = API().Marshal(v)
	return
}

func Unmarshal(data []byte, v interface{}) (err error) {
	err = API().Unmarshal(data, v)
	return
}

func UnsafeMarshal(v interface{}) []byte {
	p, err := API().Marshal(v)
	if err != nil {
		panic("json marshal object in unsafe mode failed")
		return nil
	}
	return p
}

func UnsafeUnmarshal(data []byte, v interface{}) {
	err := API().Unmarshal(data, v)
	if err != nil {
		panic("json unmarshal object in unsafe mode failed")
		return
	}
	return
}

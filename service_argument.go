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

package fns

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
)

func NewArgument(v interface{}) (arg Argument, err error) {
	if v == nil {
		err = errors.BadRequest("fns.Argument: new with nil pointer value")
		return
	}
	var p []byte
	switch v.(type) {
	case []byte:
		p = v.([]byte)
		if !json.Validate(p) {
			err = errors.BadRequest("fns.Argument: new with invalid json")
			return
		}
	default:
		p, err = json.Marshal(v)
		if err != nil {
			err = errors.BadRequest("fns.Argument: new with invalid value")
			return
		}
	}
	arg = &argument{}
	decodeErr := arg.UnmarshalJSON(p)
	if decodeErr != nil {
		err = errors.BadRequest("fns.Argument: new with invalid json")
		return
	}
	return
}

type argument []byte

func (arg argument) MarshalJSON() (data []byte, err error) {
	if arg == nil {
		err = errors.BadRequest("fns.Argument: MarshalJSON on nil pointer")
		return
	}
	if !json.Validate(arg) {
		err = errors.BadRequest("fns.Argument: MarshalJSON on invalid data")
		return
	}
	data = arg
	return
}

func (arg *argument) UnmarshalJSON(data []byte) (err error) {
	if arg == nil {
		err = errors.BadRequest("fns.Argument: UnmarshalJSON on nil pointer")
		return
	}
	if !json.Validate(data) {
		err = errors.BadRequest("fns.Argument: UnmarshalJSON on invalid data")
		return
	}
	*arg = append((*arg)[0:0], data...)
	return
}

func (arg *argument) As(v interface{}) (err error) {
	decodeErr := json.Unmarshal(*arg, v)
	if decodeErr != nil {
		err = errors.BadRequest("fns.Argument: As failed")
	}
	return
}
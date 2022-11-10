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

package service

import (
	"github.com/aacfactory/copier"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/json"
)

type Argument interface {
	json.Marshaler
	json.Unmarshaler
	As(v interface{}) (err errors.CodeError)
}

func EmptyArgument() (arg Argument) {
	arg = NewArgument(&Empty{})
	return
}

func NewArgument(v interface{}) (arg Argument) {
	if v == nil {
		arg = EmptyArgument()
		return
	}
	arg = &argument{
		value: v,
	}
	return
}

type argument struct {
	value interface{}
}

func (arg *argument) MarshalJSON() (data []byte, err error) {
	switch arg.value.(type) {
	case []byte:
		value := arg.value.([]byte)
		if !json.Validate(value) {
			err = errors.Warning("fns: type of argument is not json bytes").WithMeta("scope", "argument")
			return
		}
		data = value
		break
	case json.RawMessage:
		data = arg.value.(json.RawMessage)
		break
	default:
		data, err = json.Marshal(arg.value)
		if err != nil {
			err = errors.Warning("fns: encode argument to json failed").WithMeta("scope", "argument").WithCause(err)
			return
		}
	}
	return
}

func (arg *argument) UnmarshalJSON(data []byte) (err error) {
	arg.value = json.RawMessage(data)
	return
}

func (arg *argument) As(v interface{}) (err errors.CodeError) {
	if arg.value == nil {
		return
	}
	switch arg.value.(type) {
	case *Empty, struct{}:
		break
	case []byte:
		value := arg.value.([]byte)
		if json.Validate(value) {
			decodeErr := json.Unmarshal(value, v)
			if decodeErr != nil {
				err = errors.Warning("fns: decode argument failed").WithMeta("scope", "argument").WithCause(decodeErr)
				return
			}
		} else {
			cpErr := commons.CopyInterface(v, arg.value)
			if cpErr != nil {
				err = errors.Warning("fns: decode argument failed").WithMeta("scope", "argument").WithCause(cpErr)
				return
			}
		}
		break
	case json.RawMessage:
		value := arg.value.(json.RawMessage)
		decodeErr := json.Unmarshal(value, v)
		if decodeErr != nil {
			err = errors.Warning("fns: decode argument failed").WithMeta("scope", "argument").WithCause(decodeErr)
			return
		}
		break
	default:
		cpErr := commons.CopyInterface(v, arg.value)
		if cpErr != nil {
			cpErr = copier.Copy(v, arg.value)
			if cpErr != nil {
				err = errors.Warning("fns: decode argument failed").WithMeta("scope", "argument").WithCause(cpErr)
			}
			return
		}
	}
	return
}

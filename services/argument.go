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

package services

import (
	"bytes"
	stdjson "encoding/json"
	"github.com/aacfactory/copier"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/json"
	"github.com/cespare/xxhash/v2"
	"strconv"
)

type Argument interface {
	json.Marshaler
	As(v interface{}) (err error)
	HashCode() (code []byte)
}

func EmptyArgument() (arg Argument) {
	arg = NewArgument(empty)
	return
}

func NewArgument(v interface{}) (arg Argument) {
	if v == nil {
		arg = EmptyArgument()
		return
	}
	p, ok := v.([]byte)
	if ok {
		if bytes.Equal(null, p) {
			v = nil
		}
	}
	arg = argument{
		value: v,
	}
	return
}

type argument struct {
	value interface{}
}

func (arg argument) MarshalJSON() (data []byte, err error) {
	if arg.value == nil {
		data = null
		return
	}
	switch arg.value.(type) {
	case Empty, *Empty:
		data = null
		break
	case []byte:
		value := arg.value.([]byte)
		if !json.Validate(value) {
			data, err = json.Marshal(arg.value)
			return
		}
		data = value
		break
	case json.RawMessage:
		data = arg.value.(json.RawMessage)
		break
	case stdjson.RawMessage:
		data = arg.value.(stdjson.RawMessage)
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

func (arg argument) As(v interface{}) (err error) {
	if arg.value == nil {
		return
	}
	switch arg.value.(type) {
	case Empty, *Empty, struct{}:
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
			cpErr := objects.CopyInterface(v, arg.value)
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
	case stdjson.RawMessage:
		value := arg.value.(stdjson.RawMessage)
		decodeErr := json.Unmarshal(value, v)
		if decodeErr != nil {
			err = errors.Warning("fns: decode argument failed").WithMeta("scope", "argument").WithCause(decodeErr)
			return
		}
		break
	default:
		cpErr := objects.CopyInterface(v, arg.value)
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

func (arg argument) HashCode() (code []byte) {
	if arg.value == nil {
		return
	}
	var p []byte
	switch arg.value.(type) {
	case Empty, *Empty, struct{}:
		return
	case []byte:
		p = arg.value.([]byte)
		break
	case json.RawMessage:
		p = arg.value.(json.RawMessage)
		break
	case stdjson.RawMessage:
		p = arg.value.(stdjson.RawMessage)
		break
	default:
		p, _ = json.Marshal(arg.value)
		break
	}
	code = bytex.FromString(strconv.FormatUint(xxhash.Sum64(p), 16))
	return
}

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

package objects

import (
	stdjson "encoding/json"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"reflect"
)

func New(src interface{}) Object {
	if src == nil {
		return object{}
	}
	s, ok := src.(Object)
	if ok {
		return s
	}
	return object{
		value: src,
	}
}

type Object interface {
	Valid() (ok bool)
	TransformTo(dst any) (err error)
	json.Marshaler
}

type object struct {
	value any
}

func (obj object) Valid() (ok bool) {
	if obj.value == nil {
		return
	}
	o, isObject := obj.value.(Object)
	if isObject {
		ok = o.Valid()
		return
	}
	ok = true
	return
}

func (obj object) TransformTo(dst interface{}) (err error) {
	if dst == nil {
		err = errors.Warning("fns: transform object failed").WithCause(fmt.Errorf("dst is nil"))
		return
	}
	if !obj.Valid() {
		return
	}
	o, isObject := obj.value.(Object)
	if isObject {
		err = o.TransformTo(dst)
		if err != nil {
			err = errors.Warning("fns: transform object failed").WithCause(err)
			return
		}
		return
	}

	dpv := reflect.ValueOf(dst)
	if dpv.Kind() != reflect.Ptr {
		err = errors.Warning("fns: transform object failed").WithCause(fmt.Errorf("type of dst is not pointer"))
		return
	}
	bytes, isBytes := obj.value.([]byte)
	isJson := isBytes && json.Validate(bytes) &&
		(dpv.Elem().Type().Kind() != reflect.Slice || !(dpv.Elem().Type().Kind() == reflect.Slice && dpv.Elem().Type().Elem().Kind() == reflect.Uint8))
	if isJson {
		err = json.Unmarshal(bytes, dst)
		if err == nil {
			return
		}
	}
	sv := reflect.ValueOf(obj.value)
	dv := reflect.Indirect(dpv)
	if sv.Kind() == reflect.Ptr {
		if sv.IsNil() {
			return
		}
		sv = sv.Elem()
	}
	if sv.IsValid() && sv.Type().AssignableTo(dv.Type()) {
		dv.Set(sv)
		return
	}
	if dv.Kind() == sv.Kind() && sv.Type().ConvertibleTo(dv.Type()) {
		dv.Set(sv.Convert(dv.Type()))
		return
	}
	err = errors.Warning("fns: transform object failed").WithCause(fmt.Errorf("type of dst is not matched"))
	return
}

func (obj object) MarshalJSON() (data []byte, err error) {
	if !obj.Valid() {
		data = json.NullBytes
		return
	}
	switch obj.value.(type) {
	case struct{}, *struct{}:
		data = json.NullBytes
		break
	case []byte:
		value := obj.value.([]byte)
		if !json.Validate(value) {
			data, err = json.Marshal(obj.value)
			return
		}
		data = value
		break
	case json.RawMessage:
		data = obj.value.(json.RawMessage)
		break
	case stdjson.RawMessage:
		data = obj.value.(stdjson.RawMessage)
		break
	default:
		data, err = json.Marshal(obj.value)
		if err != nil {
			err = errors.Warning("fns: encode object to json bytes failed").WithCause(err)
			return
		}
	}
	return
}

func Value[T any](obj Object) (v T, err error) {
	o, ok := obj.(object)
	if ok {
		v, ok = o.value.(T)
		if ok {
			return
		}
		err = obj.TransformTo(&v)
		return
	}
	err = obj.TransformTo(&v)
	return
}

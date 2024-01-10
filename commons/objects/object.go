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
	"fmt"
	"github.com/aacfactory/errors"
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
	Unmarshal(dst any) (err error)
	Value() (v any)
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

func (obj object) Value() (v any) {
	if obj.value == nil {
		return
	}
	o, isObject := obj.value.(Object)
	if isObject {
		v = o.Value()
		return
	}
	v = obj.value
	return
}

func (obj object) Unmarshal(dst interface{}) (err error) {
	if dst == nil {
		err = errors.Warning("fns: unmarshal object failed").WithCause(fmt.Errorf("dst is nil"))
		return
	}
	if !obj.Valid() {
		return
	}
	o, isObject := obj.value.(Object)
	if isObject {
		err = o.Unmarshal(dst)
		if err != nil {
			err = errors.Warning("fns: unmarshal object failed").WithCause(err)
			return
		}
		return
	}

	dpv := reflect.ValueOf(dst)
	if dpv.Kind() != reflect.Ptr {
		err = errors.Warning("fns: unmarshal object failed").WithCause(fmt.Errorf("type of dst is not pointer"))
		return
	}

	// copy
	sv := reflect.ValueOf(obj.value)
	st := sv.Type()
	dv := reflect.Indirect(dpv)
	dt := dv.Type()
	if sv.Kind() == reflect.Ptr {
		if sv.IsNil() {
			return
		}
		sv = sv.Elem()
	}
	if sv.IsValid() && st.AssignableTo(dt) {
		dv.Set(sv)
		return
	}
	if dv.Kind() == sv.Kind() && st.ConvertibleTo(dt) {
		dv.Set(sv.Convert(dt))
		return
	}
	if dv.Type().Kind() == reflect.Interface && dv.CanSet() {
		if st.Implements(dt) {
			dv.Set(sv)
			return
		}
		if sv.CanAddr() && sv.Addr().Type().Implements(dt) {
			dv.Set(sv.Addr())
			return
		}
	}
	err = errors.Warning("fns: unmarshal object failed").WithCause(fmt.Errorf("type of dst is not matched"))
	return
}

func Value[T any](obj Object) (v T, err error) {
	o, ok := obj.(object)
	if ok {
		v, ok = o.value.(T)
		if ok {
			return
		}
		err = obj.Unmarshal(&v)
		return
	}
	err = obj.Unmarshal(&v)
	return
}

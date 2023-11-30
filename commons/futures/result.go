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

package futures

import (
	"fmt"
	"github.com/aacfactory/json"
	"reflect"
)

type Result interface {
	json.Marshaler
	Valid() (ok bool)
	TransformTo(dst interface{}) (err error)
}

type result struct {
	value interface{}
}

func (r result) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.value)
}

func (r result) Valid() (ok bool) {
	if r.value == nil {
		return
	}
	rr, matched := r.value.(Result)
	if matched {
		ok = rr.Valid()
		return
	}
	ok = true
	return
}

func (r result) TransformTo(dst interface{}) (err error) {
	if dst == nil {
		return
	}
	if !r.Valid() {
		return
	}
	rr, matched := r.value.(Result)
	if matched {
		err = rr.TransformTo(dst)
		return
	}
	dpv := reflect.ValueOf(dst)
	if dpv.Kind() != reflect.Ptr {
		err = fmt.Errorf("copy failed for type of dst is not ptr")
		return
	}
	sv := reflect.ValueOf(r.value)
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
	err = fmt.Errorf("scan failed for type is not matched")
	return
}

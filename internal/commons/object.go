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

package commons

import (
	"fmt"
	"reflect"
)

func CopyInterface(dst interface{}, src interface{}) (err error) {
	if dst == nil {
		err = fmt.Errorf("copy failed for dst is nil")
		return
	}
	dpv := reflect.ValueOf(dst)
	if dpv.Kind() != reflect.Ptr {
		err = fmt.Errorf("copy failed for type of dst is not ptr")
		return
	}
	sv := reflect.ValueOf(src)
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
	err = fmt.Errorf("copy failed for type is not matched")
	return
}

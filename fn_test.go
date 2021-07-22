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

package fns_test

import (
	"fmt"
	"github.com/aacfactory/fns"
	"reflect"
	"testing"
)

type ArgA struct {
	Id    string
	Names []string
	File  fns.FnFile
	Files []fns.FnFile
}

func TestFnArguments_Scan(t *testing.T) {

	a := &ArgA{}

	_type := reflect.TypeOf(a)

	elemType := _type.Elem()

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		fmt.Println(field.Type.String())
		fmt.Println(field.Type == reflect.TypeOf(fns.FnFile{}), field.Type == reflect.TypeOf([]fns.FnFile{}))
	}

	fmt.Println("xxx", reflect.TypeOf([]string{}) == reflect.TypeOf(make([]string, 0, 1)))

}

// @fn address.1
// @tag t1 t2
func FnA(fc fns.FnContext, arg ArgA) (err error) {

	return
}

func FnAHttpProxy(fc fns.FnContext, arguments fns.Arguments, tags ...string) (result interface{}, err error) {
	arg := &ArgA{}
	scanErr := arguments.Scan(arg)
	if scanErr != nil {
		err = scanErr
		return
	}

	// options := fns.DeliveryOptions{} add tags

	reply := fc.Eventbus().Request("address.1", arg)

	requestErr := reply.Get(&result)
	if requestErr != nil {
		err = requestErr
		return
	}

	return
}

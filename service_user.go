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
	"fmt"
	"github.com/aacfactory/json"
	"strconv"
)

func newUser(authorization []byte) (u User) {
	u = &user{
		authorization: authorization,
		attributes:    json.NewObject(),
		principals:    json.NewObject(),
	}
	return
}

type user struct {
	authorization []byte
	attributes    *json.Object
	principals    *json.Object
}

func (u *user) Exists() (ok bool) {
	ok = !u.principals.Empty()
	return
}

func (u *user) Id() (id UserId) {
	value := json.RawMessage(make([]byte, 0, 1))
	if u.Principals().Contains("sub") {
		_ = u.Principals().Get("sub", &value)
	}
	if u.Attributes().Contains("id") {
		_ = u.Attributes().Get("id", &value)
	}
	if len(value) == 0 {
		id = &userId{
			value: "",
		}
		return
	}
	content := ""
	if value[0] == '"' {
		content = string(value[1 : len(value)-1])
	} else {
		content = string(value)
	}
	id = &userId{
		value: content,
	}
	return
}

func (u *user) Authorization() (authorization []byte, has bool) {
	authorization = u.authorization
	has = u.authorization != nil && len(u.authorization) > 0
	return
}

func (u *user) SetAuthorization(authorization []byte) {
	u.authorization = authorization
	return
}

func (u *user) Principals() (principals *json.Object) {
	principals = u.principals
	return
}

func (u *user) Attributes() (attributes *json.Object) {
	attributes = u.attributes
	return
}

func (u *user) String() (value string) {
	value = fmt.Sprintf("User: {authorization: %s, principals: %s, attributes: %s}", string(u.authorization), string(u.principals.Raw()), string(u.attributes.Raw()))
	return
}

type userId struct {
	value string
}

func (u *userId) Int() (v int) {
	v, _ = strconv.Atoi(u.value)
	return
}

func (u *userId) String() (v string) {
	v = u.value
	return
}

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
	"github.com/aacfactory/fns/json"
)

type UserStore interface {
	Put(user User) (err error)
	Contains(user User) (has bool)
	Remove(user User) (err error)
}

type AuthorizationEncoder interface {
	Build(config Config)
	Encode(user User) (token []byte, err error)
	Decode(token []byte, user User) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

// todo: 吊销 user store，然后使用JTI的标准进行吊销，
type User interface {
	Exists() (ok bool)
	Id() (id string)
	Principals() (principal *json.Object)
	Attributes() (attributes *json.Object)
	Encode() (value []byte, err error)
	Save() (err error)
	Revoke() (err error)
	String() (value string)
}

func newUser(encoder AuthorizationEncoder) User {
	return &authorizationUser{
		attributes: json.NewObject(),
		principals: json.NewObject(),
		encoder:    encoder,
	}
}

type authorizationUser struct {
	attributes *json.Object
	principals *json.Object
	encoder    AuthorizationEncoder
}

func (u *authorizationUser) Exists() (ok bool) {
	ok = !u.principals.Empty() || !u.attributes.Empty()
	return
}

func (u *authorizationUser) Id() (id string) {
	if u.Principals().Contains("sub") {
		subErr := u.Principals().Get("sub", &id)
		if subErr == nil {
			return
		}
	}
	if u.Attributes().Contains("id") {
		idErr := u.Principals().Get("id", &id)
		if idErr == nil {
			return
		}
	}
	return
}

func (u *authorizationUser) Attributes() (attributes *json.Object) {
	attributes = u.attributes
	return
}

func (u *authorizationUser) Principals() (principals *json.Object) {
	principals = u.principals
	return
}

func (u *authorizationUser) Encode() (value []byte, err error) {
	if u.encoder != nil {
		value, err = u.encoder.Encode(u)
	}
	return
}

func (u *authorizationUser) Save() (err error) {
	return
}

func (u *authorizationUser) Revoke() (err error) {
	return
}

func (u *authorizationUser) String() (value string) {
	value = fmt.Sprintf("User: {principals: %s, attributes: %s}", string(u.principals.Raw()), string(u.attributes.Raw()))
	return
}

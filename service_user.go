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
)

func newUser() (u User) {
	u = &user{
		attributes: json.NewObject(),
		principals: json.NewObject(),
	}
	return
}

type user struct {
	attributes *json.Object
	principals *json.Object
	auth       Authorizations
}

func (u *user) Exists() (ok bool) {
	ok = !u.principals.Empty()
	return
}

func (u *user) Id() (id string) {
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

func (u *user) Principals() (principals *json.Object) {
	principals = u.principals
	return
}

func (u *user) Attributes() (attributes *json.Object) {
	attributes = u.attributes
	return
}

func (u *user) Encode() (value []byte, err error) {
	if u.auth == nil {
		err = fmt.Errorf("user Encode failed for no Authorizations contains")
	}
	value, err = u.auth.Encode(u)
	return
}

func (u *user) Active(ctx Context) (err error) {
	if u.auth == nil {
		err = fmt.Errorf("user Active failed for no Authorizations contains")
	}
	err = u.auth.Active(ctx, u)
	return
}

func (u *user) Revoke(ctx Context) (err error) {
	if u.auth == nil {
		err = fmt.Errorf("user Revoke failed for no Authorizations contains")
	}
	err = u.auth.Revoke(ctx, u)
	return
}

func (u *user) String() (value string) {
	value = fmt.Sprintf("User: {principals: %s, attributes: %s}", string(u.principals.Raw()), string(u.attributes.Raw()))
	return
}

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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
)

func newUser(authorization []byte, authorizations Authorizations, permissions Permissions) (u User) {
	u = &user{
		authorization:  authorization,
		attributes:     json.NewObject(),
		principals:     json.NewObject(),
		authorizations: authorizations,
		permissions:    permissions,
	}
	return
}

type user struct {
	authorization  []byte
	attributes     *json.Object
	principals     *json.Object
	authorizations Authorizations
	permissions    Permissions
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

func (u *user) Authorization() (authorization []byte, has bool) {
	authorization = u.authorization
	has = u.authorization != nil && len(u.authorization) > 0
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

func (u *user) CheckAuthorization() (err errors.CodeError) {
	token, has := u.Authorization()
	if !has {
		err = errors.Unauthorized("fns User: check authorization failed for authorization was not found")
		return
	}
	decodeErr := u.authorizations.Decode(token, u)
	if decodeErr != nil {
		err = errors.Unauthorized("fns User: check authorization failed for decode authorization failed").WithCause(decodeErr)
		return
	}
	return
}

func (u *user) CheckPermissions(ctx Context, namespace string, fn string) (err errors.CodeError) {
	permissionErr := u.permissions.Validate(ctx, namespace, fn, u)
	if permissionErr != nil {
		err = errors.Forbidden("fns User: check permissions failed").WithCause(permissionErr)
		return
	}
	return
}

func (u *user) EncodeToAuthorization() (value []byte, err error) {
	if u.authorizations == nil {
		err = fmt.Errorf("fns User: encode failed for no Authorizations set, please setup services Authorizations")
		return
	}
	value, err = u.authorizations.Encode(u)
	if err == nil {
		u.authorization = value
	}
	return
}

func (u *user) Active(ctx Context) (err error) {
	if u.authorizations == nil {
		err = fmt.Errorf("fns User: active failed for no Authorizations set, please setup services Authorizations")
		return
	}
	err = u.authorizations.Active(ctx, u)
	return
}

func (u *user) Revoke(ctx Context) (err error) {
	if u.authorizations == nil {
		err = fmt.Errorf("fns User: revoke failed for no Authorizations set, please setup services Authorizations")
		return
	}
	err = u.authorizations.Revoke(ctx, u)
	return
}

func (u *user) String() (value string) {
	value = fmt.Sprintf("User: {authorization: %s, principals: %s, attributes: %s}", string(u.authorization), string(u.principals.Raw()), string(u.attributes.Raw()))
	return
}

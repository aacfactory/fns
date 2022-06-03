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

package authorizations

import (
	"github.com/aacfactory/fns"
	"github.com/aacfactory/json"
	"time"
)

type Token interface {
	Id() (id string)
	NotBefore() (date time.Time)
	NotAfter() (date time.Time)
	User() (id string, attr *json.Object)
	Encode() (p []byte, err error)
}

type TokenEncoding interface {
	Build(env fns.Environments) (err error)
	Encode(id string, attributes *json.Object) (token Token, err error)
	Decode(authorization []byte) (token Token, err error)
}

type tokenEncodingComponent struct {
	encoding TokenEncoding
}

func (e *tokenEncodingComponent) Name() (name string) {
	name = "encoding"
	return
}

func (e *tokenEncodingComponent) Build(env fns.Environments) (err error) {
	err = e.encoding.Build(env)
	return
}

func (e *tokenEncodingComponent) Encode(id string, attributes *json.Object) (token Token, err error) {
	token, err = e.encoding.Encode(id, attributes)
	return
}

func (e *tokenEncodingComponent) Decode(authorization []byte) (token Token, err error) {
	token, err = e.encoding.Decode(authorization)
	return
}

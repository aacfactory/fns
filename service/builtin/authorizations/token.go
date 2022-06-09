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
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"time"
)

type Token interface {
	Id() (id string)
	NotBefore() (date time.Time)
	NotAfter() (date time.Time)
	User() (id string, attr *json.Object)
	Encode() (p []byte, err error)
}

type TokenEncodingOptions struct {
	Log    logs.Logger
	Config configuares.Config
}

type TokenEncoding interface {
	Build(options TokenEncodingOptions) (err error)
	Encode(id string, attributes *json.Object) (token Token, err error)
	Decode(authorization []byte) (token Token, err error)
}

type tokenEncodingComponent struct {
	encoding TokenEncoding
}

func (component *tokenEncodingComponent) Name() (name string) {
	name = "encoding"
	return
}

func (component *tokenEncodingComponent) Build(options service.ComponentOptions) (err error) {
	config, hasConfig := options.Config.Node("encoding")
	if !hasConfig {
		err = errors.Warning("fns: build authorizations token encoding failed").WithCause(fmt.Errorf("there is no encoding node in authorizations config node"))
		return
	}
	err = component.encoding.Build(TokenEncodingOptions{
		Log:    options.Log,
		Config: config,
	})
	return
}

func (component *tokenEncodingComponent) Encode(id string, attributes *json.Object) (token Token, err error) {
	token, err = component.encoding.Encode(id, attributes)
	return
}

func (component *tokenEncodingComponent) Decode(authorization []byte) (token Token, err error) {
	token, err = component.encoding.Decode(authorization)
	return
}

func (component *tokenEncodingComponent) Close() {
}

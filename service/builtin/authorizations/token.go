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
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/internal/secret"
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
	Bytes() (p []byte)
}

type TokenEncodingOptions struct {
	Log    logs.Logger
	Config configures.Config
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
	err = component.encoding.Build(TokenEncodingOptions{
		Log:    options.Log,
		Config: options.Config,
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

type defaultToken struct {
	Id_        string          `json:"id"`
	NotBefore_ time.Time       `json:"notBefore"`
	NotAfter_  time.Time       `json:"notAfter"`
	Uid        string          `json:"uid"`
	Attributes json.RawMessage `json:"attributes"`
	p          []byte
}

func (t *defaultToken) Id() (id string) {
	id = t.Id_
	return
}

func (t *defaultToken) NotBefore() (date time.Time) {
	date = t.NotBefore_
	return
}

func (t *defaultToken) NotAfter() (date time.Time) {
	date = t.NotAfter_
	return
}

func (t *defaultToken) User() (id string, attr *json.Object) {
	id, attr = t.Uid, json.NewObjectFromBytes(t.Attributes)
	return
}

func (t *defaultToken) Bytes() (p []byte) {
	p = t.p
	return
}

func createDefaultTokenEncoding() TokenEncoding {
	return &DefaultTokenEncoding{}
}

type DefaultTokenEncoding struct {
	expires time.Duration
}

func (encoding *DefaultTokenEncoding) Build(options TokenEncodingOptions) (err error) {
	expireMinutes := 0
	_, expireMinutesGetErr := options.Config.Get("expireMinutes", &expireMinutes)
	if expireMinutesGetErr != nil {
		err = errors.Warning("fns: default token encoding build failed").WithCause(expireMinutesGetErr).WithMeta("component", "DefaultTokenEncoding")
		return
	}
	if expireMinutes < 1 {
		expireMinutes = 24 * 60
	}
	encoding.expires = time.Duration(expireMinutes) * time.Minute
	return
}

func (encoding *DefaultTokenEncoding) Encode(id string, attributes *json.Object) (t Token, err error) {
	v := &defaultToken{
		Id_:        uid.UID(),
		NotBefore_: time.Now(),
		NotAfter_:  time.Now().Add(encoding.expires),
		Uid:        id,
		Attributes: attributes.Raw(),
	}
	p, encodeErr := json.Marshal(v)
	if encodeErr != nil {
		err = errors.Warning("fns: default token encoding failed").WithCause(encodeErr).WithMeta("component", "DefaultTokenEncoding")
		return
	}
	signature := secret.Sign(p)
	v.p = []byte(fmt.Sprintf("%s.%s", base64.StdEncoding.EncodeToString(p), base64.StdEncoding.EncodeToString(signature)))
	t = v
	return
}

func (encoding *DefaultTokenEncoding) Decode(authorization []byte) (token Token, err error) {
	if authorization == nil || len(authorization) < 6 || bytes.Index(authorization, []byte("Fns ")) != 0 {
		err = errors.Warning("fns: invalid authorization").WithMeta("component", "DefaultTokenEncoding")
		return
	}
	raw := authorization[4:]
	items := bytes.Split(raw, []byte{'.'})
	if len(items) != 2 {
		err = errors.Warning("fns: invalid authorization").WithMeta("component", "DefaultTokenEncoding")
		return
	}
	if !secret.Verify(items[0], items[1]) {
		err = errors.Warning("fns: invalid authorization").WithMeta("component", "DefaultTokenEncoding")
		return
	}
	v := &defaultToken{}
	decodeErr := json.Unmarshal(items[0], v)
	if decodeErr != nil {
		err = errors.Warning("fns: invalid authorization").WithMeta("component", "DefaultTokenEncoding").WithCause(decodeErr)
		return
	}
	v.p = raw
	return
}

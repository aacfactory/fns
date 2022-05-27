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
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
)

var authorizationsRetrieverMap = make(map[string]AuthorizationsRetriever)

type AuthorizationsRetriever func(config configuares.Raw) (authorizations Authorizations, err error)

func RegisterAuthorizationsRetriever(kind string, retriever AuthorizationsRetriever) {
	authorizationsRetrieverMap[kind] = retriever
}

type Authorizations interface {
	Encode(ctx Context, claims interface{}) (token []byte, err errors.CodeError)
	Decode(ctx Context, token []byte) (err errors.CodeError)
}

type fakeAuthorizations struct{}

func (auth *fakeAuthorizations) Encode(_ Context, _ interface{}) (token []byte, err errors.CodeError) {
	err = errors.Warning("fns Authorizations: authorizations was not enabled, please use fns.RegisterAuthorizationsRetriever() to setup")
	return
}

func (auth *fakeAuthorizations) Decode(_ Context, _ []byte) (err errors.CodeError) {
	err = errors.Warning("fns Authorizations: authorizations was not enabled, please use fns.RegisterAuthorizationsRetriever() to setup")
	return
}
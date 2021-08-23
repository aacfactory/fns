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
	"encoding/base64"
	"fmt"
	"strings"
)

// +-------------------------------------------------------------------------------------------------------------------+

type AuthorizationCredentialsRetriever func(header RequestHeader) (credentials AuthCredentials, has bool)

var BearerAuthorizationCredentialsRetriever = func(header RequestHeader) (credentials AuthCredentials, has bool) {
	value, exist := header.Get(httpHeaderAuthorization)
	if !exist {
		return
	}
	credentials = NewTokenCredentials(value)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type AuthorizationValidator interface {
	Build(config Config, log Logs)
	Handle(authorization string) (user User, err CodeError)
}


// +-------------------------------------------------------------------------------------------------------------------+

type AuthCredentials interface {
	ToHttpAuthorization() (v string)
}

// 以下实现转到 contrib 中

func NewUsernamePasswordCredentials(username string, password string) *UsernamePasswordCredentials {
	return &UsernamePasswordCredentials{
		username: username,
		password: password,
	}
}

func NewUsernamePasswordCredentialsFromHttpAuthorization(authorization string) (credentials *UsernamePasswordCredentials, err error) {
	if authorization == "" {
		err = fmt.Errorf("create username password credentials from http authorization failed, authorization is empty")
		return
	}
	spc := strings.IndexByte(authorization, ' ')
	if "basic" != strings.ToLower(authorization[0:spc]) {
		err = fmt.Errorf("create username password credentials from http authorization failed, authorization is invalid")
		return
	}
	decoded, decodeErr := base64.StdEncoding.DecodeString(authorization[spc+1:])
	if decodeErr != nil {
		err = fmt.Errorf("create username password credentials from http authorization failed, authorization is invalid, %v", decodeErr)
		return
	}
	items := strings.Split(string(decoded), ":")
	credentials = &UsernamePasswordCredentials{
		username: items[0],
		password: items[1],
	}
	return
}

type UsernamePasswordCredentials struct {
	username string
	password string
}

func (credentials *UsernamePasswordCredentials) Username() string {
	return credentials.username
}

func (credentials *UsernamePasswordCredentials) Password() string {
	return credentials.password
}

func (credentials *UsernamePasswordCredentials) ToHttpAuthorization() (v string) {
	v = fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(credentials.username+":"+credentials.password)))
	return
}

func NewTokenCredentials(token string, scopes ...string) *TokenCredentials {
	if scopes == nil {
		scopes = make([]string, 0, 1)
	}
	return &TokenCredentials{
		token:  token,
		scopes: scopes,
	}
}

func NewTokenCredentialsFromHttpAuthorization(authorization string) (credentials *TokenCredentials, err error) {
	if authorization == "" {
		err = fmt.Errorf("create token credentials from http authorization failed, authorization is empty")
		return
	}
	spc := strings.IndexByte(authorization, ' ')
	if "Bearer" != strings.ToLower(authorization[0:spc]) {
		err = fmt.Errorf("create token credentials from http authorization failed, authorization is invalid")
		return
	}
	credentials = &TokenCredentials{
		token:  authorization[spc+1:],
		scopes: make([]string, 0, 1),
	}
	return
}

type TokenCredentials struct {
	token  string
	scopes []string
}

func (credentials *TokenCredentials) Token() string {
	return credentials.token
}

func (credentials *TokenCredentials) Scopes() []string {
	return credentials.scopes
}

func (credentials *TokenCredentials) AddScopes(scopes ...string) {
	if scopes == nil || len(scopes) == 0 {
		return
	}
	storedScopes := credentials.scopes
	if storedScopes == nil {
		storedScopes = make([]string, 0, 1)
	}
	for _, scope := range scopes {
		exists := false
		for _, stored := range storedScopes {
			if stored == scope {
				exists = true
				break
			}
		}
		if !exists {
			storedScopes = append(storedScopes, scope)
		}
	}
	credentials.scopes = storedScopes
}

func (credentials *TokenCredentials) ToHttpAuthorization() (v string) {
	v = fmt.Sprintf("Bearer %s", credentials.token)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+


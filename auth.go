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
	"time"
)

type AuthCredentials interface {
	ToHttpAuthorization() (v string)
}

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
	if "bearer" != strings.ToLower(authorization[0:spc]) {
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

type User interface {
	Expired() (expired bool)
	Attributes() (attributes UserAttributes)
	Principal() (principal UserPrincipal)
}

type UserAttributes interface {
	Contains(key string) (has bool)
	Put(key string, value interface{})
	Get(key string, value interface{}) (err error)
	GetString(key string) (value string, err error)
	GetStringArray(key string) (value []string, err error)
	GetBool(key string) (value bool, err error)
	GetInt(key string) (value int, err error)
	GetInt32(key string) (value int32, err error)
	GetInt64(key string) (value int64, err error)
	GetFloat32(key string) (value float32, err error)
	GetFloat64(key string) (value float64, err error)
	GetTime(key string) (value time.Time, err error)
	GetDuration(key string) (value time.Duration, err error)
}

func newFnsUserAttributes() UserAttributes {
	return &fnsUserAttributes{
		data: NewJsonObject(),
	}
}

type fnsUserAttributes struct {
	data *JsonObject
}

func (attr *fnsUserAttributes) Contains(key string) (has bool) {
	has = attr.data.Contains(key)
	return
}

func (attr *fnsUserAttributes) Put(key string, value interface{}) {
	err := attr.data.Put(key, value)
	if err != nil {
		panic(fmt.Errorf("user attributes put %s %v failed", key, value))
	}
	return
}

func (attr *fnsUserAttributes) Get(key string, value interface{}) (err error) {
	err = attr.data.Get(key, value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetString(key string) (value string, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetStringArray(key string) (value []string, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetBool(key string) (value bool, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetInt(key string) (value int, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetInt32(key string) (value int32, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetInt64(key string) (value int64, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetFloat32(key string) (value float32, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetFloat64(key string) (value float64, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetTime(key string) (value time.Time, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

func (attr *fnsUserAttributes) GetDuration(key string) (value time.Duration, err error) {
	err = attr.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user attributes put %s failed", key)
	}
	return
}

type UserPrincipal interface {
	Contains(key string) (has bool)
	Put(key string, value interface{})
	Get(key string, value interface{}) (err error)
	GetString(key string) (value string, err error)
	GetStringArray(key string) (value []string, err error)
	GetBool(key string) (value bool, err error)
	GetInt(key string) (value int, err error)
	GetInt32(key string) (value int32, err error)
	GetInt64(key string) (value int64, err error)
	GetFloat32(key string) (value float32, err error)
	GetFloat64(key string) (value float64, err error)
	GetTime(key string) (value time.Time, err error)
	GetDuration(key string) (value time.Duration, err error)
}

func newFnsUserPrincipal() UserPrincipal {
	return &fnsUserPrincipal{
		data: NewJsonObject(),
	}
}

type fnsUserPrincipal struct {
	data *JsonObject
}

func (principal *fnsUserPrincipal) Contains(key string) (has bool) {
	has = principal.data.Contains(key)
	return
}

func (principal *fnsUserPrincipal) Put(key string, value interface{}) {
	err := principal.data.Put(key, value)
	if err != nil {
		panic(fmt.Errorf("user principal put %s %v failed", key, value))
	}
	return
}

func (principal *fnsUserPrincipal) Get(key string, value interface{}) (err error) {
	err = principal.data.Get(key, value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetString(key string) (value string, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetStringArray(key string) (value []string, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetBool(key string) (value bool, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetInt(key string) (value int, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetInt32(key string) (value int32, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetInt64(key string) (value int64, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetFloat32(key string) (value float32, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetFloat64(key string) (value float64, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetTime(key string) (value time.Time, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

func (principal *fnsUserPrincipal) GetDuration(key string) (value time.Duration, err error) {
	err = principal.data.Get(key, &value)
	if err != nil {
		err = fmt.Errorf("user principal put %s failed", key)
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

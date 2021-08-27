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
	sc "context"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"time"
)

type Context interface {
	sc.Context
	RequestId() (id string)
	Authorization() (value []byte)
	User() (user User)
	Meta() (meta ContextMeta)
	Log() (log logs.Logger)
	ServiceProxy(namespace string) (proxy ServiceProxy, err error)
	Timeout() (has bool)
	Validate(v interface{}) (err errors.CodeError)
}

// +-------------------------------------------------------------------------------------------------------------------+

type ContextMeta interface {
	Exists(key string) (has bool)
	Put(key string, value interface{})
	Get(key string, value interface{}) (err error)
	GetString(key string) (value string, has bool)
	GetInt(key string) (value int, has bool)
	GetInt32(key string) (value int32, has bool)
	GetInt64(key string) (value int64, has bool)
	GetFloat32(key string) (value float32, has bool)
	GetFloat64(key string) (value float64, has bool)
	GetBool(key string) (value bool, has bool)
	GetTime(key string) (value time.Time, has bool)
	GetDuration(key string) (value time.Duration, has bool)
	Encode() (value []byte)
	Decode(value []byte) (ok bool)
}

// +-------------------------------------------------------------------------------------------------------------------+

// Services
// 管理 Service，具备 Service 的注册与发现
type Services interface {
	Build(config ServicesConfig) (err error)
	Mount(service Service) (err error)
	Exist(namespace string) (ok bool)
	Description(namespace string) (description []byte)
	DecodeAuthorization(ctx Context, value []byte) (err errors.CodeError)
	PermissionAllow(ctx Context, namespace string, fn string) (err errors.CodeError)
	Request(ctx Context, namespace string, fn string, argument Argument) (result Result)
	Close()
}

// +-------------------------------------------------------------------------------------------------------------------+

// Service
// 管理 Fn 的服务
type Service interface {
	Namespace() (namespace string)
	Build(config configuares.Config) (err error)
	Description() (description []byte)
	Handle(context Context, fn string, argument Argument) (result interface{}, err errors.CodeError)
	Close() (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Argument interface {
	json.Marshaler
	json.Unmarshaler
	As(v interface{}) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Result interface {
	Succeed(v interface{})
	Failed(err errors.CodeError)
	Get(ctx sc.Context, v interface{}) (err errors.CodeError)
}

// +-------------------------------------------------------------------------------------------------------------------+

type ServiceRequestHeader interface {
	Get(name string) (value []byte, has bool)
}

// +-------------------------------------------------------------------------------------------------------------------+

var authorizationsRetrieverMap = make(map[string]AuthorizationsRetriever)

type AuthorizationsRetriever func(config configuares.Raw) (authorizations Authorizations, err error)

// RegisterAuthorizationsRetriever
// 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterAuthorizationsRetriever(kind string, retriever AuthorizationsRetriever) {
	authorizationsRetrieverMap[kind] = retriever
}

type Authorizations interface {
	Encode(user User) (token []byte, err error)
	Decode(token []byte, user User) (err error)
	Active(user User) (err error)
	Revoke(user User) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type User interface {
	Exists() (ok bool)
	Id() (id string)
	Principals() (principal *json.Object)
	Attributes() (attributes *json.Object)
	Encode() (value []byte, err error)
	Active() (err error)
	Revoke() (err error)
	String() (value string)
}

// +-------------------------------------------------------------------------------------------------------------------+

var permissionsRetrieverMap = make(map[string]PermissionsRetriever)

type PermissionsRetriever func(config configuares.Raw) (permission Permissions, err error)

// RegisterPermissionsRetriever
// 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterPermissionsRetriever(kind string, retriever PermissionsRetriever) {
	permissionsRetrieverMap[kind] = retriever
}

type Permissions interface {
	Validate(ctx Context, namespace string, fnName string, user User) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type ServiceProxy interface {
	Request(ctx Context, fn string, argument Argument) (result Result)
}

// +-------------------------------------------------------------------------------------------------------------------+

var serviceDiscoveryRetrieverMap = map[string]ServiceDiscoveryRetriever{
	"default": standaloneServiceDiscoveryRetriever,
}

type ServiceDiscoveryRetriever func(options ServiceDiscoveryOption) (discovery ServiceDiscovery, err error)

// RegisterServiceDiscoveryRetriever
// 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterServiceDiscoveryRetriever(kind string, retriever ServiceDiscoveryRetriever) {
	if serviceDiscoveryRetrieverMap == nil {
		serviceDiscoveryRetrieverMap = make(map[string]ServiceDiscoveryRetriever)
	}
	serviceDiscoveryRetrieverMap[kind] = retriever
}

type ServiceDiscoveryOption struct {
	ServerId string
	Address  string
	Config   []byte
}

type ServiceDiscovery interface {
	Publish(service Service) (err error)
	IsLocal(namespace string) (ok bool)
	Proxy(namespace string) (proxy ServiceProxy, err errors.CodeError)
	Close()
}

// +-------------------------------------------------------------------------------------------------------------------+

type Empty struct{}

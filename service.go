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
	Authorization() (value string)
	User() (user User)
	Meta() (meta ContextMeta)
	Log() (log logs.Logger)
	ServiceProxy(namespace string) (proxy ServiceProxy, err error)
	Timeout() (has bool)
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

// ServiceCenter
// 管理 Service，具备 Service 的注册与发现
type ServiceCenter interface {
	Build(config configuares.Config) (err error)
	Mount(service Service) (err error)
	Request(ctx Context, namespace string, fn string, argument Argument) (result Result)
	Close()
}

// +-------------------------------------------------------------------------------------------------------------------+

// Service
// todo: fnc 生成 service，然后service的函数是代理实际写的函数，并注入参数、用户等前置条件，以及缓存、事件队列等pipeline式的函数。
type Service interface {
	Namespace() (namespace string)
	Build(config configuares.Config) (err error)
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
	Get(v interface{}) (err errors.CodeError)
}

// +-------------------------------------------------------------------------------------------------------------------+

type ServiceRequestConfig struct {
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	RequestBodyMaxSize int
}

type ServiceRequestConfigBuilder interface {
	Build(namespace string, fnName string, header ServiceRequestHeader) (config ServiceRequestConfig)
}

// +-------------------------------------------------------------------------------------------------------------------+

type ServiceRequestHeader interface {
	Get(name string) (value []byte, has bool)
}

// +-------------------------------------------------------------------------------------------------------------------+

var authorizationsRetrieverMap map[string]AuthorizationsRetriever = nil

type AuthorizationsRetriever func(options AuthorizationsOption) (authorizations Authorizations, err error)

// RegisterAuthorizationsRetriever
// 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterAuthorizationsRetriever(kind string, retriever AuthorizationsRetriever) {
	if authorizationsRetrieverMap == nil {
		authorizationsRetrieverMap = make(map[string]AuthorizationsRetriever)
	}
	authorizationsRetrieverMap[kind] = retriever
}

type AuthorizationsOption struct {
	Config    []byte
	UserStore UserStore
}

type Authorizations interface {
	Encode(user User) (token []byte, err error)
	Decode(token []byte, user User) (err error)
	Active(user User) (err error)
	Revoke(user User) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

var userStoreRetrieverMap map[string]UserStoreRetriever = nil

type UserStoreRetriever func(options UserStoreOption) (store UserStore, err error)

// RegisterUserStoreRetriever
// 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterUserStoreRetriever(kind string, retriever UserStoreRetriever) {
	if userStoreRetrieverMap == nil {
		userStoreRetrieverMap = make(map[string]UserStoreRetriever)
	}
	userStoreRetrieverMap[kind] = retriever
}

type UserStoreOption struct {
	Config []byte
}

type UserStore interface {
	Save(user User) (err error)
	Contains(user User) (has bool)
	Remove(user User) (err error)
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

var permissionsRetrieverMap map[string]PermissionsRetriever = nil

type PermissionsRetriever func(options PermissionsOption) (permission Permissions, err error)

// RegisterPermissionsRetriever
// 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterPermissionsRetriever(kind string, retriever PermissionsRetriever) {
	if permissionsRetrieverMap == nil {
		permissionsRetrieverMap = make(map[string]PermissionsRetriever)
	}
	permissionsRetrieverMap[kind] = retriever
}

type PermissionsOption struct {
	Config []byte
}

type Permissions interface {
	Validate(ctx Context, namespace string, fnName string, user User) (err errors.CodeError)
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
	ServerId  string
	Address   string
	ClientTLS ClientTLS
	Config    []byte
}

type ServiceDiscovery interface {
	Publish(service Service) (err error)
	Proxy(namespace string) (proxy ServiceProxy, err error)
	Close()
}

// +-------------------------------------------------------------------------------------------------------------------+

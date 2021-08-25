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
// todo: fnc 生成 service，然后service的函数是代理实际写的函数，并注入参数、用户等前置条件，以及缓存、事件队列等pipeline式的函数。
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
	Get(v interface{}) (err errors.CodeError)
}

// +-------------------------------------------------------------------------------------------------------------------+

type ServiceRequestHeader interface {
	Get(name string) (value []byte, has bool)
}

// +-------------------------------------------------------------------------------------------------------------------+

var authorizationsRetrieverMap map[string]AuthorizationsRetriever = nil

type AuthorizationsRetriever func(config configuares.Raw) (authorizations Authorizations, err error)

// RegisterAuthorizationsRetriever
// 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterAuthorizationsRetriever(kind string, retriever AuthorizationsRetriever) {
	if authorizationsRetrieverMap == nil {
		authorizationsRetrieverMap = make(map[string]AuthorizationsRetriever)
	}
	authorizationsRetrieverMap[kind] = retriever
}

type Authorizations interface {
	Encode(user User) (token []byte, err error)
	Decode(token []byte, user User) (err error)
	Active(user User) (err error)
	Revoke(user User) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

// todo: move into jwt Authorizations
var userStoreRetrieverMap map[string]UserStoreRetriever = nil

type UserStoreRetriever func(config configuares.Raw) (store UserStore, err error)

// RegisterUserStoreRetriever
// 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterUserStoreRetriever(kind string, retriever UserStoreRetriever) {
	if userStoreRetrieverMap == nil {
		userStoreRetrieverMap = make(map[string]UserStoreRetriever)
	}
	userStoreRetrieverMap[kind] = retriever
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

type PermissionsRetriever func(config configuares.Raw) (permission Permissions, err error)

// RegisterPermissionsRetriever
// 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterPermissionsRetriever(kind string, retriever PermissionsRetriever) {
	if permissionsRetrieverMap == nil {
		permissionsRetrieverMap = make(map[string]PermissionsRetriever)
	}
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

func NewServiceDescription(namespace string, description string) (d *ServiceDescription) {
	d = &ServiceDescription{}
	return
}

// ServiceDescription open api struct
type ServiceDescription struct {
	// todo json.RawMessage from swagger
	Info ServiceDescriptionInfo
	// todo json.RawMessage from swagger
	Paths map[string]map[string]ServiceDescriptionPath
	// todo json.RawMessage from swagger
	Definitions map[string]ServiceDescriptionDefinition
}

func (d *ServiceDescription) AppendPath(uri string, operationId string, summary string, description string, needAuth bool, paramName string, resultName string) {
	p := ServiceDescriptionPath{}
	// todo
	d.Paths[uri] = map[string]ServiceDescriptionPath{"post": p}
}

func (d *ServiceDescription) AppendDefinition(name string, definition ServiceDescriptionDefinition) {
	d.Definitions[name] = definition
}

type ServiceDescriptionInfo struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	// todo sjson.Put
	Version string `json:"version,omitempty"`
}

type ServiceDescriptionPath struct {
	OperationId string
	Summary     string
	Description string
	Consumes    []string
	Produces    []string
	Parameters  []ServiceDescriptionPathParam
	Responses   map[string]ServiceDescriptionPathResponse
}

type ServiceDescriptionPathParam struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	In          string            `json:"in,omitempty"`
	Required    bool              `json:"required"`
	Type        string            `json:"type,omitempty"`
	Schema      map[string]string `json:"schema,omitempty"`
}

type ServiceDescriptionPathResponse struct {
	Description string            `json:"description,omitempty"`
	Schema      map[string]string `json:"schema,omitempty"`
}

type ServiceDescriptionDefinition struct {
	// object array
	Type       string                                          `json:"type,omitempty"`
	Required   []string                                        `json:"required,omitempty"`
	Items      map[string]string                               `json:"items,omitempty"`
	Properties map[string]ServiceDescriptionDefinitionProperty `json:"properties,omitempty"`
}

type ServiceDescriptionDefinitionProperty struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Ref         string `json:"$ref,omitempty"`
	Items       map[string]string
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Maximum     int         `json:"maximum,omitempty"`
	Minimum     int         `json:"minimum,omitempty"`
	MaxLength   int         `json:"maxLength,omitempty"`
	MinLength   int         `json:"minLength,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
}

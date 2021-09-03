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
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"sync"
	"time"
)

type AppRuntime interface {
	PublicAddress() (address string)
	Log() (log logs.Logger)
	Validate(v interface{}) (err errors.CodeError)
	ServiceProxy(ctx Context, namespace string) (proxy ServiceProxy, err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Context interface {
	sc.Context
	RequestId() (id string)
	User() (user User)
	Meta() (meta ContextMeta)
	Timeout() (has bool)
	App() (app AppRuntime)
}

// +-------------------------------------------------------------------------------------------------------------------+

// context meta keys
const (
	// ServiceProxyAddress 指定Namespace服务代理的Address，value格式为 namespace/address
	serviceExactProxyMetaKeyPrefix = "exact_proxy"
)

type ContextMeta interface {
	Exists(key string) (has bool)
	Put(key string, value interface{})
	Get(key string, value interface{}) (err error)
	Remove(key string)
	GetString(key string) (value string, has bool)
	GetInt(key string) (value int, has bool)
	GetInt32(key string) (value int32, has bool)
	GetInt64(key string) (value int64, has bool)
	GetFloat32(key string) (value float32, has bool)
	GetFloat64(key string) (value float64, has bool)
	GetBool(key string) (value bool, has bool)
	GetTime(key string) (value time.Time, has bool)
	GetDuration(key string) (value time.Duration, has bool)
	SetExactProxyService(namespace string, address string)
	GetExactProxyService() (namespace string, address string, has bool)
	DelExactProxyService(namespace string)
	Encode() (value []byte)
	Decode(value []byte) (ok bool)
}

// +-------------------------------------------------------------------------------------------------------------------+

// Service
// 管理 Fn 的服务
type Service interface {
	Namespace() (namespace string)
	Internal() (internal bool)
	Build(config configuares.Config) (err error)
	Description() (description []byte)
	Handle(context Context, fn string, argument Argument) (result interface{}, err errors.CodeError)
	Close() (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Argument interface {
	json.Marshaler
	json.Unmarshaler
	As(v interface{}) (err errors.CodeError)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Result interface {
	Succeed(v interface{})
	Failed(err errors.CodeError)
	Get(ctx sc.Context, v interface{}) (err errors.CodeError)
}

// +-------------------------------------------------------------------------------------------------------------------+

var authorizationsRetrieverMap = make(map[string]AuthorizationsRetriever)

type AuthorizationsRetriever func(config configuares.Raw) (authorizations Authorizations, err error)

func RegisterAuthorizationsRetriever(kind string, retriever AuthorizationsRetriever) {
	authorizationsRetrieverMap[kind] = retriever
}

type Authorizations interface {
	Encode(user User) (token []byte, err error)
	Decode(token []byte, user User) (err error)
	IsActive(ctx Context, user User) (ok bool)
	Active(ctx Context, user User) (err error)
	Revoke(ctx Context, user User) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type User interface {
	Exists() (ok bool)
	Id() (id string)
	Principals() (principal *json.Object)
	Attributes() (attributes *json.Object)
	Authorization() (authorization []byte, has bool)
	CheckAuthorization() (err errors.CodeError)
	CheckPermissions(ctx Context, namespace string, fn string) (err errors.CodeError)
	EncodeToAuthorization() (value []byte, err error)
	IsActive(ctx Context) (ok bool)
	Active(ctx Context) (err error)
	Revoke(ctx Context) (err error)
	String() (value string)
}

// +-------------------------------------------------------------------------------------------------------------------+

var permissionsRetrieverMap = make(map[string]PermissionsRetriever)

type PermissionsRetriever func(config configuares.Raw) (permission Permissions, err error)

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

type ServiceDiscoveryRetriever func(option ServiceDiscoveryOption) (discovery ServiceDiscovery, err error)

func RegisterServiceDiscoveryRetriever(kind string, retriever ServiceDiscoveryRetriever) {
	if serviceDiscoveryRetrieverMap == nil {
		serviceDiscoveryRetrieverMap = make(map[string]ServiceDiscoveryRetriever)
	}
	serviceDiscoveryRetrieverMap[kind] = retriever
}

type ServiceDiscoveryOption struct {
	Address            string
	HttpClientPoolSize int
	Config             configuares.Raw
}

type Registration struct {
	Id        string `json:"id"`
	Namespace string `json:"namespace,omitempty"`
	Address   string `json:"address"`
	Reversion int64  `json:"-"`
}

func (r Registration) Key() (key string) {
	key = r.Id
	return
}

func NewRegistrations() (registrations *Registrations) {
	registrations = &Registrations{
		r: commons.NewRing(),
	}
	return
}

type Registrations struct {
	r *commons.Ring
}

func (r *Registrations) Next() (v *Registration, has bool) {
	p := r.r.Next()
	if p == nil {
		return
	}
	v, has = p.(*Registration)
	return
}

func (r *Registrations) Append(v Registration) {
	r.r.Append(v)
	return
}

func (r *Registrations) Remove(v Registration) {
	r.r.Remove(v)
	return
}

func (r *Registrations) Size() (size int) {
	size = r.r.Size()
	return
}

func NewRegistrationsManager() (manager *RegistrationsManager) {
	manager = &RegistrationsManager{
		mutex:           sync.RWMutex{},
		problemCh:       make(chan *Registration, 512),
		stopListenCh:    make(chan struct{}, 1),
		registrationMap: make(map[string]*Registrations),
	}
	manager.ListenProblemChan()
	return
}

type RegistrationsManager struct {
	mutex           sync.RWMutex
	problemCh       chan *Registration
	stopListenCh    chan struct{}
	registrationMap map[string]*Registrations
}

func (manager *RegistrationsManager) Registrations() (v []Registration) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	v = make([]Registration, 0, 1)
	for _, registrations := range manager.registrationMap {
		for i := 0; i < registrations.Size(); i++ {
			registration, has := registrations.Next()
			if has {
				v = append(v, *registration)
			}
		}
	}
	return
}

func (manager *RegistrationsManager) Append(registration Registration) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	registrations, has := manager.registrationMap[registration.Namespace]
	if !has {
		registrations = NewRegistrations()
	}
	registrations.Append(registration)
	return
}

func (manager *RegistrationsManager) Remove(registration Registration) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	registrations, has := manager.registrationMap[registration.Namespace]
	if !has {
		return
	}
	registrations.Remove(registration)
	if registrations.Size() == 0 {
		delete(manager.registrationMap, registration.Namespace)
	}
	return
}

func (manager *RegistrationsManager) Get(namespace string) (registration *Registration, exists bool) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	registrations, has := manager.registrationMap[namespace]
	if !has {
		return
	}
	registration, exists = registrations.Next()
	return
}

func (manager *RegistrationsManager) ProblemChan() (ch chan<- *Registration) {
	ch = manager.problemCh
	return
}

func (manager *RegistrationsManager) ListenProblemChan() {
	go func(manager *RegistrationsManager) {
		for {
			stopped := false
			select {
			case <-manager.stopListenCh:
				stopped = true
				break
			case r, ok := <-manager.problemCh:
				if !ok {
					stopped = true
					break
				}
				manager.Remove(*r)
			}
			if stopped {
				break
			}
		}
	}(manager)
	return
}

func (manager *RegistrationsManager) Close() {
	manager.stopListenCh <- struct{}{}
	return
}

type ServiceDiscovery interface {
	Publish(service Service) (err error)
	IsLocal(namespace string) (ok bool)
	Proxy(ctx Context, namespace string) (proxy ServiceProxy, err errors.CodeError)
	Close()
}

// +-------------------------------------------------------------------------------------------------------------------+

type Empty struct{}

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
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/singleflight"
	"sync"
	"time"
)

type AppRuntime interface {
	ClusterMode() (ok bool)
	PublicAddress() (address string)
	Log() (log logs.Logger)
	Validate(v interface{}) (err errors.CodeError)
	ServiceProxy(ctx Context, namespace string) (proxy ServiceProxy, err error)
	ServiceMeta() (meta ServiceMeta)
	Authorizations() (authorizations Authorizations)
	Permissions() (permissions Permissions)
	HttpClient() (client HttpClient)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Context interface {
	sc.Context
	InternalRequested() (ok bool)
	RequestId() (id string)
	User() (user User)
	Meta() (meta ContextMeta)
	Timeout() (has bool)
	App() (app AppRuntime)
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
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
	SetExactProxyServiceAddress(namespace string, address string)
	GetExactProxyServiceAddress(namespace string) (address string, has bool)
	DelExactProxyServiceAddress(namespace string)
	Encode() (value []byte)
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewServiceMeta() ServiceMeta {
	return make(map[string]interface{})
}

type ServiceMeta map[string]interface{}

func (meta ServiceMeta) Get(key string) (v interface{}, has bool) {
	v, has = meta[key]
	return
}

func (meta ServiceMeta) Set(key string, value interface{}) {
	meta[key] = value
	return
}

func (meta ServiceMeta) merge(o ServiceMeta) {
	if o == nil || len(o) == 0 {
		return
	}
	for k, v := range o {
		meta[k] = v
	}
	return
}

type ServiceOption struct {
	MetaBuilder ServiceMetaBuilder
}

type ServiceMetaBuilder func(config configuares.Config) (meta ServiceMeta, err error)

func fakeServiceMetaBuilder(config configuares.Config) (meta ServiceMeta, err error) {
	return
}

func NewAbstractService() AbstractService {
	return NewAbstractServiceWithOption(ServiceOption{
		MetaBuilder: fakeServiceMetaBuilder,
	})
}

func NewAbstractServiceWithOption(option ServiceOption) AbstractService {
	return AbstractService{
		option: option,
		meta:   make(map[string]interface{}),
		group:  new(singleflight.Group),
	}
}

type AbstractService struct {
	option ServiceOption
	meta   ServiceMeta
	group  *singleflight.Group
}

func (s AbstractService) Meta() (v ServiceMeta) {
	v = s.meta
	return
}

func (s AbstractService) Build(config configuares.Config) (err error) {
	if config != nil && s.option.MetaBuilder != nil {
		meta, metaErr := s.option.MetaBuilder(config)
		if metaErr != nil {
			err = metaErr
			return metaErr
		}
		s.meta.merge(meta)
	}
	return
}

func (s AbstractService) HandleInGroup(ctx Context, fn string, arg Argument, handle func() (v interface{}, err errors.CodeError)) (v interface{}, err errors.CodeError) {
	key := fmt.Sprintf("%s:%s", fn, arg.Hash(ctx))
	v0, err0, _ := s.group.Do(key, func() (v interface{}, err error) {
		v, err = handle()
		return
	})
	s.group.Forget(key)
	if err0 != nil {
		err = err0.(errors.CodeError)
		return
	}
	v = v0
	return
}

// Service
// 管理 Fn 的服务
type Service interface {
	Namespace() (namespace string)
	Internal() (internal bool)
	Build(config configuares.Config) (err error)
	Document() (doc *ServiceDocument)
	Handle(context Context, fn string, argument Argument) (result interface{}, err errors.CodeError)
	Shutdown() (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Argument interface {
	json.Marshaler
	json.Unmarshaler
	As(v interface{}) (err errors.CodeError)
	Hash(ctx Context) (p string)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Result interface {
	Succeed(v interface{})
	Failed(err errors.CodeError)
	Get(ctx sc.Context, v interface{}) (err errors.CodeError)
}

// +-------------------------------------------------------------------------------------------------------------------+

type User interface {
	Exists() (ok bool)
	Id() (id string)
	Principals() (principal *json.Object)
	Attributes() (attributes *json.Object)
	Authorization() (authorization []byte, has bool)
	SetAuthorization(authorization []byte)
	String() (value string)
}

// +-------------------------------------------------------------------------------------------------------------------+

var authorizationsRetrieverMap = make(map[string]AuthorizationsRetriever)

type AuthorizationsRetriever func(config configuares.Raw) (authorizations Authorizations, err error)

func RegisterAuthorizationsRetriever(kind string, retriever AuthorizationsRetriever) {
	authorizationsRetrieverMap[kind] = retriever
}

type Authorizations interface {
	Encode(ctx Context) (token []byte, err errors.CodeError)
	Decode(ctx Context, token []byte) (err errors.CodeError)
}

// +-------------------------------------------------------------------------------------------------------------------+

var permissionsDefinitionsLoaderRetrieverMap = make(map[string]PermissionsDefinitionsLoaderRetriever)

type PermissionsDefinitionsLoaderRetriever func(config configuares.Raw) (loader PermissionsDefinitionsLoader, err error)

func RegisterPermissionsDefinitionsLoaderRetriever(kind string, retriever PermissionsDefinitionsLoaderRetriever) {
	permissionsDefinitionsLoaderRetrieverMap[kind] = retriever
}

// Permissions
// 基于RBAC的权限控制器
// 角色：角色树，控制器不存储用户的角色。
// 资源：fn
// 控制：是否可以使用（不可以使用优先于可以使用）
type Permissions interface {
	// Validate 验证当前 context 中 user 对 fn 的权限
	Validate(ctx Context, namespace string, fn string) (err errors.CodeError)
	// SaveUserRoles 将角色保存到 当前 context 的 user attributes 中
	SaveUserRoles(ctx Context, roles ...string) (err errors.CodeError)
}

type PermissionsDefinitions struct {
	data map[string]map[string]bool
}

func (d *PermissionsDefinitions) Add(namespace string, fn string, role string, accessible bool) {
	if namespace == "" || fn == "" || role == "" {
		return
	}
	if d.data == nil {
		d.data = make(map[string]map[string]bool)
	}
	key := fmt.Sprintf("%s:%s", namespace, fn)
	g, has := d.data[key]
	if !has {
		g = make(map[string]bool)
	}
	g[role] = accessible
	d.data[key] = g
}
func (d *PermissionsDefinitions) Accessible(namespace string, fn string, roles []string) (accessible bool) {
	if namespace == "" || fn == "" || d.data == nil || len(d.data) == 0 {
		return
	}
	key := fmt.Sprintf("%s:%s", namespace, fn)
	g, has := d.data[key]
	if !has {
		accessible = false
		return
	}
	_, all := g["*"]
	if all {
		accessible = true
		return
	}
	not := false
	n := 0
	for _, role := range roles {
		x, hasRole := g[role]
		if !hasRole {
			continue
		}
		if !x {
			not = true
			break
		}
		n++
	}
	if not {
		accessible = false
		return
	}
	accessible = n > 0
	return
}

// PermissionsDefinitionsLoader
// 存储权限设定的加载器
type PermissionsDefinitionsLoader interface {
	Load() (definitions *PermissionsDefinitions, err errors.CodeError)
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
	Address     string
	HttpClients *HttpClients
	Config      configuares.Raw
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

func newRegistrationsManager(clients *HttpClients) (manager *RegistrationsManager) {
	manager = &RegistrationsManager{
		clients:         clients,
		problemCh:       make(chan *Registration, 512),
		stopListenCh:    make(chan struct{}, 1),
		registrationMap: sync.Map{},
	}
	manager.ListenProblemChan()
	return
}

type RegistrationsManager struct {
	clients         *HttpClients
	problemCh       chan *Registration
	stopListenCh    chan struct{}
	registrationMap sync.Map
}

func (manager *RegistrationsManager) Registrations() (v []Registration) {
	v = make([]Registration, 0, 1)
	manager.registrationMap.Range(func(_, value interface{}) bool {
		registrations := value.(*Registrations)
		for i := 0; i < registrations.Size(); i++ {
			registration, has := registrations.Next()
			if has {
				v = append(v, *registration)
			}
		}
		return true
	})
	return
}

func (manager *RegistrationsManager) Append(registration Registration) {

	var registrations *Registrations
	value, has := manager.registrationMap.Load(registration.Namespace)
	if has {
		registrations = value.(*Registrations)
	} else {
		registrations = NewRegistrations()
	}
	registrations.Append(registration)
	manager.registrationMap.Store(registration.Namespace, registrations)
	return
}

func (manager *RegistrationsManager) Remove(registration Registration) {
	value, has := manager.registrationMap.Load(registration.Namespace)
	if !has {
		return
	}

	registrations := value.(*Registrations)
	registrations.Remove(registration)
	if registrations.Size() == 0 {
		manager.registrationMap.Delete(registration.Namespace)
	}

	return
}

func (manager *RegistrationsManager) Get(namespace string) (registration *Registration, exists bool) {
	value, has := manager.registrationMap.Load(namespace)
	if !has {
		return
	}

	registrations := value.(*Registrations)
	registration, exists = registrations.Next()
	return
}

func (manager *RegistrationsManager) ProblemChan() (ch chan<- *Registration) {
	ch = manager.problemCh
	return
}

func (manager *RegistrationsManager) CheckRegistration(registration Registration) (ok bool) {
	client := manager.clients.next()
	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	request.URI().SetHost(registration.Address)
	request.URI().SetPath(healthCheckPath)
	request.Header.SetMethodBytes(get)

	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(response)

	err := client.DoTimeout(request, response, 2*time.Second)
	if err != nil {
		return
	}

	ok = response.StatusCode() == 200
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
				if !manager.CheckRegistration(*r) {
					manager.Remove(*r)
				}
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
	Registrations() (registrations map[string]*Registration)
	Close()
}

// +-------------------------------------------------------------------------------------------------------------------+

// Empty
// @description Empty
type Empty struct{}

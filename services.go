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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/workers"
	"strings"
	"sync"
	"time"
)

const (
	fnRequestWorkHandleAction = "+"
)

type services struct {
	wp             workers.Workers
	descriptions   map[string][]byte
	internals      map[string]int64
	discovery      ServiceDiscovery
	authorizations Authorizations
	permissions    Permissions
	clients        *HttpClients
	payloads       sync.Pool
	version        string
	clusterMode    bool
}

func (s *services) Build(config ServicesConfig) (err error) {
	concurrency := config.concurrency
	if concurrency < 1 {
		concurrency = workers.DefaultConcurrency
	}
	maxIdleTimeSecond := time.Duration(config.MaxIdleTimeSecond) * time.Second
	if maxIdleTimeSecond == 0 {
		maxIdleTimeSecond = 10 * time.Second
	}

	wp, wpErr := workers.New(s, workers.WithConcurrency(concurrency), workers.WithMaxIdleTime(maxIdleTimeSecond))
	if wpErr != nil {
		err = fmt.Errorf("fns Services: build failed, %v", wpErr)
		return
	}
	s.wp = wp

	// internals
	s.internals = make(map[string]int64)

	// http clients
	httpClientPoolSize := config.HttpClientPoolSize
	if httpClientPoolSize < 1 {
		httpClientPoolSize = 10
	}
	s.clients = NewHttpClients(httpClientPoolSize)

	// discovery
	var discoveryRetriever ServiceDiscoveryRetriever
	discoveryConfig := config.Discovery
	if !discoveryConfig.Enable {
		discoveryRetriever = standaloneServiceDiscoveryRetriever
		s.clusterMode = false
	} else {
		kind := strings.TrimSpace(discoveryConfig.Kind)
		if kind == "" {
			err = fmt.Errorf("fns Services: Build failed for invalid kind")
			return
		}
		has := false
		discoveryRetriever, has = serviceDiscoveryRetrieverMap[kind]
		if !has || discoveryRetriever == nil {
			err = fmt.Errorf("fns Services: build failed for %s kind was not register, please use fns.RegisterServiceDiscoveryRetriever() to register retriever", kind)
			return
		}
		if config.address == "" {
			err = fmt.Errorf("fns Services: build failed for %s kind, public host and public port was not set", kind)
			return
		}
		s.clusterMode = true
	}

	discovery, discoveryErr := discoveryRetriever(ServiceDiscoveryOption{
		Address:     config.address,
		Config:      discoveryConfig.Config,
		HttpClients: s.clients,
	})

	if discoveryErr != nil {
		err = fmt.Errorf("fns Services: build failed, %v", discoveryErr)
		return
	}
	s.discovery = discovery

	// authorizations
	if config.Authorization.Enable {
		kind := strings.TrimSpace(config.Authorization.Kind)
		authorizationsRetriever, has := authorizationsRetrieverMap[kind]
		if !has || authorizationsRetriever == nil {
			err = fmt.Errorf("fns Services: build failed for %s kind Authorization was not register, please use fns.RegisterAuthorizationsRetriever() to register retriever", kind)
			return
		}
		authorizations, authorizationsErr := authorizationsRetriever(config.Authorization.Config)
		if authorizationsErr != nil {
			err = fmt.Errorf("fns Services: build failed, %v", authorizationsErr)
			return
		}
		s.authorizations = authorizations
	} else {
		// fake authorizations
		s.authorizations = &fakeAuthorizations{}
	}

	// permissions
	if config.Permission.Enable {
		loader := strings.TrimSpace(config.Permission.Loader)
		if loader == "" {
			err = fmt.Errorf("fns Services: build failed for %s PermissionsDefinitionsLoader was not register, please use fns.RegisterPermissionsDefinitionsLoaderRetriever() to register retriever", loader)
			return
		}
		permissionsDefinitionsLoaderRetriever, has := permissionsDefinitionsLoaderRetrieverMap[loader]
		if !has || permissionsDefinitionsLoaderRetriever == nil {
			err = fmt.Errorf("fns Services: build failed for %s PermissionsDefinitionsLoader was not register, please use fns.RegisterPermissionsRetriever() to register retriever", loader)
			return
		}
		permissionsDefinitionsLoader, permissionsDefinitionsLoaderErr := permissionsDefinitionsLoaderRetriever(config.Permission.Config)
		if permissionsDefinitionsLoaderErr != nil {
			err = fmt.Errorf("fns Services: build failed, %v", permissionsDefinitionsLoaderErr)
			return
		}
		permissions, permissionsErr := newRbacPermissions(permissionsDefinitionsLoader)
		if permissionsErr != nil {
			err = fmt.Errorf("fns Services: build failed, %v", permissionsErr)
			return
		}
		s.permissions = permissions
	} else {
		// fake permissions
		s.permissions = &fakePermissions{}
	}

	// payloads
	s.payloads.New = func() interface{} {
		return &servicesRequestPayload{}
	}

	// descriptions
	s.descriptions = make(map[string][]byte)

	// version
	s.version = config.version

	s.wp.Start()

	return
}

func (s *services) ClusterMode() (ok bool) {
	ok = s.clusterMode
	return
}

func (s *services) Mount(service Service) (err error) {
	pubErr := s.discovery.Publish(service)
	if pubErr != nil {
		err = fmt.Errorf("fns Services: mount failed, %v", pubErr)
		return
	}
	description := service.Description()
	if description != nil {
		s.descriptions[service.Namespace()] = description
	}
	if service.Internal() {
		s.internals[service.Namespace()] = 0
	}
	return
}

func (s *services) Exist(namespace string) (ok bool) {
	ok = s.discovery.IsLocal(namespace)
	return
}

func (s *services) IsInternal(namespace string) (ok bool) {
	_, ok = s.internals[namespace]
	return
}

func (s *services) Authorizations() (authorizations Authorizations) {
	authorizations = s.authorizations
	return
}

func (s *services) Permissions() (p Permissions) {
	p = s.permissions
	return
}

func (s *services) Request(ctx Context, namespace string, fn string, argument Argument) (result Result) {

	if !s.discovery.IsLocal(namespace) {
		result = SyncResult()
		result.Failed(errors.NotFound(fmt.Sprintf("fns Services: %s service was not found", namespace)))
		return
	}

	payload := s.payloads.Get().(*servicesRequestPayload)
	payload.ctx = ctx
	payload.namespace = namespace
	payload.fn = fn
	payload.argument = argument
	payload.result = AsyncResult()

	result = payload.result

	if !s.wp.Execute(fnRequestWorkHandleAction, payload) {
		s.payloads.Put(payload)
		result = SyncResult()
		result.Failed(errors.New(429, "***TOO MANY REQUEST***", fmt.Sprintf("fns Services: no work unit remains for %s/%s", namespace, fn)))
		return
	}

	return
}

func (s *services) Description(namespace string) (description []byte) {
	description0, has := s.descriptions[namespace]
	if !has {
		return
	}
	obj := json.NewObjectFromBytes(description0)
	setErr := obj.Put("info.version", s.version)
	if setErr != nil {
		description = description0
		return
	}
	description = obj.Raw()
	return
}

func (s *services) Close() {
	s.discovery.Close()
	s.wp.Stop()
}

func (s *services) Handle(action string, _payload interface{}) {
	defer s.payloads.Put(_payload)

	payload := _payload.(*servicesRequestPayload)

	if action != fnRequestWorkHandleAction {
		payload.result.Failed(errors.Unavailable("not fn request"))
		return
	}

	ctx := payload.ctx

	proxy, proxyErr := s.discovery.Proxy(ctx, payload.namespace)
	if proxyErr != nil {
		payload.result.Failed(proxyErr)
		return
	}

	// request
	r := proxy.Request(ctx, payload.fn, payload.argument)
	raw := json.RawMessage{}
	err := r.Get(ctx, &raw)
	if err != nil {
		payload.result.Failed(err)
	} else {
		payload.result.Succeed(raw)
	}
}

type servicesRequestPayload struct {
	ctx       Context
	namespace string
	fn        string
	argument  Argument
	result    Result
}

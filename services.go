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
	"github.com/tidwall/sjson"
	"strings"
	"sync"
	"time"
)

const (
	fnRequestWorkHandleAction = "invoke"
)

type services struct {
	wp             workers.Workers
	descriptions   map[string][]byte
	discovery      ServiceDiscovery
	authorizations Authorizations
	permissions    Permissions
	payloads       sync.Pool
	version        string
}

func (s *services) Build(config ServicesConfig) (err error) {
	wp, wpErr := workers.New(s, workers.WithConcurrency(config.Concurrency), workers.WithMaxIdleTime(time.Duration(config.MaxIdleTimeSecond)*time.Second))
	if wpErr != nil {
		err = fmt.Errorf("fns Services: Build failed, %v", wpErr)
		return
	}
	s.wp = wp

	// discovery
	var discoveryRetriever ServiceDiscoveryRetriever
	discoveryConfig := config.Discovery
	if !discoveryConfig.Enable {
		discoveryRetriever = standaloneServiceDiscoveryRetriever
	} else {
		kind := strings.TrimSpace(discoveryConfig.Kind)
		if kind == "" {
			err = fmt.Errorf("fns Services: Build failed for invalid kind")
			return
		}
		has := false
		discoveryRetriever, has = serviceDiscoveryRetrieverMap[kind]
		if !has || discoveryRetriever == nil {
			err = fmt.Errorf("fns Services: Build failed for %s kind was not register, please use fns.RegisterServiceDiscoveryRetriever() to register retriever", kind)
			return
		}
	}

	discovery, discoveryErr := discoveryRetriever(ServiceDiscoveryOption{
		ServerId: config.serverId,
		Address:  config.address,
		Config:   discoveryConfig.Config,
	})

	if discoveryErr != nil {
		err = fmt.Errorf("fns Services: Build failed, %v", discoveryErr)
		return
	}
	s.discovery = discovery

	// authorizations
	if config.Authorization.Enable {
		kind := strings.TrimSpace(config.Authorization.Kind)
		authorizationsRetriever, has := authorizationsRetrieverMap[kind]
		if !has || authorizationsRetriever == nil {
			err = fmt.Errorf("fns Services: Build failed for %s kind Authorization was not register, please use fns.RegisterAuthorizationsRetriever() to register retriever", kind)
			return
		}
		authorizations, authorizationsErr := authorizationsRetriever(config.Authorization.Config)
		if authorizationsErr != nil {
			err = fmt.Errorf("fns Services: Build failed, %v", authorizationsErr)
			return
		}
		s.authorizations = authorizations
	}

	// permissions
	if config.Permission.Enable {
		kind := strings.TrimSpace(config.Permission.Kind)
		permissionsRetriever, has := permissionsRetrieverMap[kind]
		if !has || permissionsRetriever == nil {
			err = fmt.Errorf("fns Services: Build failed for %s kind Permissions was not register, please use fns.RegisterPermissionsRetriever() to register retriever", kind)
			return
		}
		permissions, permissionsErr := permissionsRetriever(config.Permission.Config)
		if permissionsErr != nil {
			err = fmt.Errorf("fns Services: Build failed, %v", permissionsErr)
			return
		}
		s.permissions = permissions
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

func (s *services) Mount(service Service) (err error) {
	pubErr := s.discovery.Publish(service)
	if pubErr != nil {
		err = fmt.Errorf("fns Services: Mount failed, %v", pubErr)
		return
	}
	description := service.Description()
	if description != nil {
		s.descriptions[service.Namespace()] = description
	}
	return
}

func (s *services) Exist(namespace string) (ok bool) {
	ok = s.discovery.IsLocal(namespace)
	return
}

func (s *services) DecodeAuthorization(ctx Context, value []byte) (err errors.CodeError) {
	if s.authorizations == nil {
		err = errors.NotImplemented("Services: DecodeAuthorization failed for Authorizations was not set, please use fns.RegisterAuthorizationsRetriever() to set")
		return
	}

	decodeErr := s.authorizations.Decode(value, ctx.User())
	if decodeErr != nil {
		err = errors.Unauthorized(decodeErr.Error())
		return
	}

	_ctx := ctx.(*context)
	_ctx.user.(*user).auth = s.authorizations
	_ctx.authorization = value

	return
}

func (s *services) PermissionAllow(ctx Context, namespace string, fn string) (err errors.CodeError) {
	if s.permissions == nil {
		return
	}

	notPass := s.permissions.Validate(ctx, namespace, fn, ctx.User())
	if notPass != nil {
		err = errors.Forbidden(notPass.Error())
		return
	}

	return
}

func (s *services) Request(ctx Context, namespace string, fn string, argument Argument) (result Result) {
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
		result.Failed(errors.New(429, "***TOO MANY REQUEST***", fmt.Sprintf("no work unit remains for %s/%s", namespace, fn)))
		return
	}

	return
}

func (s *services) Description(namespace string) (description []byte) {
	description0, has := s.descriptions[namespace]
	if !has {
		return
	}
	p, setErr := sjson.SetBytes(description, "info.version", s.version)
	if setErr != nil {
		description = description0
		return
	}
	description = p
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

	proxy, proxyErr := s.discovery.Proxy(payload.namespace)
	if proxyErr != nil {
		payload.result.Failed(proxyErr)
		return
	}

	ctx := WithNamespace(payload.ctx, payload.namespace)
	ctx = withDiscovery(ctx, s.discovery)

	r := proxy.Request(ctx, payload.fn, payload.argument)
	raw := json.RawMessage{}
	err := r.Get(&raw)
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

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
	"strings"
	"time"

	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"github.com/go-playground/validator/v10"
)

const (
	fnRequestWorkHandleAction = "+"
)

func newServices(serverId string, version string, publicAddress string, concurrency int, log logs.Logger, validate *validator.Validate) (v *services) {
	if concurrency < 1 {
		concurrency = workers.DefaultConcurrency
	}
	v = &services{
		concurrency:   concurrency,
		doc:           nil,
		internals:     make(map[string]int64),
		version:       version,
		serverId:      serverId,
		publicAddress: publicAddress,
		log:           log,
		validate:      validate,
	}
	return
}

type services struct {
	concurrency     int
	wp              workers.Workers
	doc             *document
	internals       map[string]int64
	discovery       ServiceDiscovery
	authorizations  Authorizations
	permissions     Permissions
	clients         *HttpClients
	version         string
	serverId        string
	clusterMode     bool
	publicAddress   string
	log             logs.Logger
	validate        *validator.Validate
	fnHandleTimeout time.Duration
}

func (s *services) buildDocument(appName string, appDesc string, appTerm string, https bool, contact *appContact, license *appLicense) {
	info := documentInfo{
		Title:          appName,
		Description:    appDesc,
		TermsOfService: appTerm,
		Contact:        nil,
		License:        nil,
		version:        s.version,
	}
	if contact != nil {
		info.Contact = &documentInfoContact{
			Name:  contact.name,
			Url:   contact.url,
			Email: contact.email,
		}
	}
	if license != nil {
		info.License = &documentInfoLicense{
			Name: license.name,
			Url:  license.url,
		}
	}
	s.doc = newDocument(info, s.publicAddress, https)
}

func (s *services) Build(config ServicesConfig) (err error) {
	// timeout
	handleTimeout := 30 * time.Second
	if config.HandleTimeoutSecond > 0 {
		handleTimeout = time.Duration(config.HandleTimeoutSecond) * time.Second
	}
	s.fnHandleTimeout = handleTimeout

	// workers
	maxIdleTimeSecond := time.Duration(config.MaxIdleTimeSecond) * time.Second
	if maxIdleTimeSecond == 0 {
		maxIdleTimeSecond = 10 * time.Second
	}

	wp, wpErr := workers.New(s, workers.WithConcurrency(s.concurrency), workers.WithMaxIdleTime(maxIdleTimeSecond))
	if wpErr != nil {
		err = fmt.Errorf("fns Services: build failed, %v", wpErr)
		return
	}
	s.wp = wp

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
		if s.publicAddress == "" {
			err = fmt.Errorf("fns Services: build failed for %s kind, public host and public port was not set", kind)
			return
		}
		s.clusterMode = true
	}

	discovery, discoveryErr := discoveryRetriever(ServiceDiscoveryOption{
		Address:     s.publicAddress,
		Config:      discoveryConfig.Config,
		HttpClients: s.clients,
	})

	if discoveryErr != nil {
		err = fmt.Errorf("fns Services: build failed, %v", discoveryErr)
		return
	}
	s.discovery = discovery
	//

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

	s.wp.Start()

	return
}

func (s *services) Mount(service Service) (err error) {
	pubErr := s.discovery.Publish(service)
	if pubErr != nil {
		err = fmt.Errorf("fns Services: mount failed, %v", pubErr)
		return
	}
	if service.Internal() {
		s.internals[service.Namespace()] = 0
	} else {
		doc := service.Document()
		if doc != nil {
			s.doc.addServiceDocument(doc)
		}
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

func (s *services) Request(isInnerRequest bool, requestId string, meta []byte, authorization []byte, namespace string, fn string, argument Argument) (result Result) {

	if !s.discovery.IsLocal(namespace) {
		result = SyncResult()
		result.Failed(errors.NotFound(fmt.Sprintf("fns Services: %s service was not found", namespace)))
		return
	}

	payload := &servicesRequestPayload{}
	payload.isInnerRequest = isInnerRequest
	payload.requestId = requestId
	payload.meta = meta
	payload.authorization = authorization
	payload.namespace = namespace
	payload.fn = fn
	payload.argument = argument
	payload.result = AsyncResult()

	result = payload.result

	if !s.wp.Execute(fnRequestWorkHandleAction, payload) {
		result = SyncResult()
		result.Failed(errors.New(429, "***TOO MANY REQUEST***", fmt.Sprintf("fns Services: no work unit remains for %s/%s", namespace, fn)))
		return
	}

	return
}

func (s *services) Close() {
	s.discovery.Close()
	s.wp.Stop()
}

func (s *services) Handle(action string, _payload interface{}) {

	payload := _payload.(*servicesRequestPayload)

	if action != fnRequestWorkHandleAction {
		payload.result.Failed(errors.Unavailable("fns Services: not fn request"))
		return
	}

	// ctx
	if !payload.isInnerRequest && s.IsInternal(payload.namespace) {
		payload.result.Failed(errors.Warning("fns Services: can not access an internal service"))
		return
	}
	timeoutCtx, cancel := sc.WithTimeout(sc.TODO(), s.fnHandleTimeout)
	ctx, ctxErr := newContext(timeoutCtx, payload.isInnerRequest, payload.requestId, payload.authorization, payload.meta, &appRuntime{
		clusterMode:    s.clusterMode,
		publicAddress:  s.publicAddress,
		appLog:         s.log,
		validate:       s.validate,
		discovery:      s.discovery,
		authorizations: s.authorizations,
		permissions:    s.permissions,
		httpClients:    s.clients,
	})
	if ctxErr != nil {
		payload.result.Failed(errors.Warning("fns Context: create context from request failed").WithCause(ctxErr))
		cancel()
		return
	}

	// proxy
	proxy, proxyErr := s.discovery.Proxy(ctx, payload.namespace)
	if proxyErr != nil {
		payload.result.Failed(proxyErr)
		cancel()
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
	cancel()
}

type servicesRequestPayload struct {
	isInnerRequest bool
	requestId      string
	meta           []byte
	authorization  []byte
	namespace      string
	fn             string
	argument       Argument
	result         Result
}

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
	"github.com/aacfactory/fns/cluster"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"net/http"
	"strings"
	"time"
)

type Endpoint interface {
	Request(ctx Context, fn string, argument Argument) (result Result)
}

type localUnitPayload struct {
	ctx      Context
	service  Service
	fn       string
	argument Argument
	result   Result
}

type localEndpoint struct {
	workerPool workers.Workers
	service    Service
}

func (endpoint *localEndpoint) Request(ctx Context, fn string, argument Argument) (result Result) {
	result = NewResult()
	ok := endpoint.workerPool.Execute("local", &localUnitPayload{
		ctx:      ctx,
		service:  endpoint.service,
		fn:       fn,
		argument: argument,
		result:   result,
	})
	if !ok {
		result.Failed(errors.NotAcceptable("fns: send request to endpoint failed for no workers").WithMeta("fns", "endpoints").WithMeta("cause", "out of workers"))
	}
	return
}

type remoteUnitPayload struct {
	ctx           Context
	exact         bool
	registration  *cluster.Registration
	registrations *cluster.Registrations
	service       string
	fn            string
	argument      Argument
	result        Result
}

type remoteEndpoint struct {
	workerPool    workers.Workers
	service       string
	exact         bool
	registration  *cluster.Registration
	registrations *cluster.Registrations
}

func (endpoint *remoteEndpoint) Request(ctx Context, fn string, argument Argument) (result Result) {
	result = NewResult()
	ok := endpoint.workerPool.Execute("remote", &remoteUnitPayload{
		ctx:           ctx,
		exact:         endpoint.exact,
		registration:  endpoint.registration,
		registrations: endpoint.registrations,
		service:       endpoint.service,
		fn:            fn,
		argument:      argument,
		result:        result,
	})
	if !ok {
		result.Failed(errors.Warning("fns: send request to endpoint failed").WithMeta("scope", "system"))
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Endpoints interface {
	Get(ctx Context, name string) (endpoint Endpoint, err errors.CodeError)
	GetExact(ctx Context, name string, registrationId string) (endpoint Endpoint, err errors.CodeError)
}

type serviceEndpointsOptions struct {
	workerMaxIdleTime time.Duration
	barrier           Barrier
	clusterManager    *cluster.Manager
}

func newEndpoints(env Environments, opt serviceEndpointsOptions) (v *serviceEndpoints, err error) {
	var registrationManager *cluster.RegistrationsManager
	if opt.clusterManager != nil {
		registrationManager = opt.clusterManager.Registrations()
	}
	handler := newEndpointHandler(env, registrationManager)
	workerPool, workerPoolErr := workers.New(handler, workers.WithConcurrency(workers.DefaultConcurrency), workers.WithMaxIdleTime(opt.workerMaxIdleTime))
	if workerPoolErr != nil {
		err = fmt.Errorf("fns: create endpoints failed for unable to create workers, %s", workerPoolErr)
		return
	}
	v = &serviceEndpoints{
		appId:         env.AppId(),
		endpoints:     make(map[string]*localEndpoint),
		registrations: registrationManager,
		workerPool:    workerPool,
		handler:       handler,
	}
	return
}

type serviceEndpoints struct {
	appId         string
	endpoints     map[string]*localEndpoint
	registrations *cluster.RegistrationsManager
	workerPool    workers.Workers
	handler       *endpointHandler
}

func (s *serviceEndpoints) Get(ctx Context, name string) (endpoint Endpoint, err errors.CodeError) {
	canAccessInternal := ctx.CanAccessInternal()
	local, got := s.endpoints[name]
	if got {
		if local.service.Internal() {
			if !canAccessInternal {
				err = errors.NotAcceptable(fmt.Sprintf("fns: can not access %s service", name)).WithMeta("scope", "endpoints")
				return
			}
		}
		endpoint = local
		return
	}
	if !canAccessInternal {
		err = errors.NotFound(fmt.Sprintf("fns: there is no %s service", name)).WithMeta("scope", "endpoints")
		return
	}
	if s.registrations == nil {
		err = errors.NotFound(fmt.Sprintf("fns: there is no %s service", name)).WithMeta("scope", "endpoints")
		return
	}
	registrations, hasRegistrations := s.registrations.GetRegistrations(name)
	if !hasRegistrations {
		return
	}
	endpoint = &remoteEndpoint{
		workerPool:    s.workerPool,
		exact:         false,
		registration:  nil,
		registrations: registrations,
		service:       name,
	}
	return
}

func (s *serviceEndpoints) GetExact(ctx Context, name string, registrationId string) (endpoint Endpoint, err errors.CodeError) {
	canAccessInternal := ctx.CanAccessInternal()
	if !canAccessInternal {
		err = errors.NotAcceptable(fmt.Sprintf("fns: can not access %s service", name)).WithMeta("scope", "endpoints")
		return
	}
	if registrationId == s.appId {
		local, got := s.endpoints[name]
		if !got {
			err = errors.NotFound(fmt.Sprintf("fns: there is no %s service", name)).WithMeta("scope", "endpoints")
			return
		}
		endpoint = local
		return
	}
	if registrationId == "" {
		local, got := s.endpoints[name]
		if got {
			endpoint = local
			return
		}
	}
	if s.registrations == nil {
		err = errors.NotFound(fmt.Sprintf("fns: there is no %s service", name)).WithMeta("scope", "endpoints")
		return
	}
	registration, hasRegistration := s.registrations.GetRegistration(name, registrationId)
	if !hasRegistration {
		err = errors.NotFound(fmt.Sprintf("fns: there is no %s service endpoint in discovery", name)).WithMeta("scope", "endpoints")
		return
	}
	endpoint = &remoteEndpoint{
		workerPool:    s.workerPool,
		exact:         true,
		registration:  registration,
		registrations: nil,
		service:       name,
	}
	return
}

func (s *serviceEndpoints) mount(service Service) {
	name := strings.TrimSpace(service.Name())
	s.endpoints[name] = &localEndpoint{
		workerPool: s.workerPool,
		service:    service,
	}
	return
}

func (s *serviceEndpoints) start() (err errors.CodeError) {
	s.workerPool.Start()
	return
}

func (s *serviceEndpoints) close() (err errors.CodeError) {
	s.workerPool.Stop()
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func newEndpointHandler(env Environments, registrations *cluster.RegistrationsManager) *endpointHandler {
	return &endpointHandler{
		log:           env.Log().With("fns", "endpoint"),
		registrations: registrations,
	}
}

type endpointHandler struct {
	log           logs.Logger
	registrations *cluster.RegistrationsManager
}

func (h *endpointHandler) Handle(action string, payload interface{}) {
	switch action {
	case "local":
		h.handleLocalAction(payload.(*localUnitPayload))
	case "remote":
		h.handleRemoteAction(payload.(*remoteUnitPayload))
	default:
		if h.log.DebugEnabled() {
			h.log.Debug().Message(fmt.Sprintf("fns: worker handle failed for action named %s is invalid", action))
		}
	}
	return
}

func (h *endpointHandler) handleLocalAction(payload *localUnitPayload) {
	service := payload.service
	fn := payload.fn
	parentCtx := payload.ctx.(*context)
	ctx := &context{
		Context:           parentCtx.Context,
		request:           parentCtx.request,
		data:              parentCtx.data,
		log:               parentCtx.runtime.Log().With("service", service.Name()).With("fn", fn),
		tracer:            parentCtx.tracer,
		serviceComponents: service.Components(),
		runtime:           parentCtx.runtime,
	}
	arg := payload.argument
	result := payload.result
	// span.Begin()
	span := ctx.tracer.StartSpan(service.Name(), fn)
	service.Handle(ctx, fn, arg, result)
	// span.End()
	span.Finish()
}

func (h *endpointHandler) handleRemoteAction(payload *remoteUnitPayload) {
	fn := payload.fn
	parentCtx := payload.ctx.(*context)
	arg := payload.argument
	result := payload.result
	if payload.exact {
		registration := payload.registration
		ctx := &context{
			Context:           parentCtx.Context,
			request:           parentCtx.request,
			data:              parentCtx.data,
			log:               parentCtx.runtime.Log().With("service", registration.Name).With("fn", fn).With("registration", fmt.Sprintf("%s:%s:%s", registration.Name, registration.Id, registration.Address)),
			tracer:            parentCtx.tracer,
			serviceComponents: nil,
			runtime:           parentCtx.runtime,
		}
		span := ctx.tracer.StartSpan(registration.Name, fn)
		span.AddTag("remote", registration.Address)
		proxyResult, proxyErr := proxy(ctx, span, registration, fn, arg)
		span.Finish()
		if proxyErr == nil {
			span.AddTag("status", "OK")
			span.AddTag("handled", "succeed")
			result.Succeed(proxyResult)
		} else {
			span.AddTag("status", proxyErr.Name())
			span.AddTag("handled", "failed")
			if proxyErr.Code() == http.StatusServiceUnavailable {
				registration.AddUnavailableTimes()
				if registration.Unavailable() {
					h.registrations.RemoveUnavailableRegistration(registration.Name, registration.Id)
				}
			}
			result.Failed(proxyErr)
		}
	} else {
		registrations := payload.registrations.Size()
		handled := false
		for i := 0; i < registrations; i++ {
			registration, hasRegistration := payload.registrations.Next()
			if !hasRegistration {
				result.Failed(errors.NotFound(fmt.Sprintf("fns: there is no %s service", payload.service)).WithMeta("scope", "endpoints"))
				handled = true
				break
			}
			ctx := &context{
				Context:           parentCtx.Context,
				request:           parentCtx.request,
				data:              parentCtx.data,
				log:               parentCtx.runtime.Log().With("service", registration.Name).With("fn", fn).With("registration", fmt.Sprintf("%s:%s:%s", registration.Name, registration.Id, registration.Address)),
				tracer:            parentCtx.tracer,
				serviceComponents: nil,
				runtime:           parentCtx.runtime,
			}
			span := ctx.tracer.StartSpan(registration.Name, fn)
			span.AddTag("remote", registration.Address)
			proxyResult, proxyErr := proxy(ctx, span, registration, fn, arg)
			span.Finish()
			if proxyErr == nil {
				span.AddTag("status", "OK")
				span.AddTag("handled", "succeed")
				result.Succeed(proxyResult)
				handled = true
				break
			} else {
				span.AddTag("status", proxyErr.Name())
				span.AddTag("handled", "failed")
				if proxyErr.Code() == http.StatusServiceUnavailable {
					registration.AddUnavailableTimes()
					if registration.Unavailable() {
						h.registrations.RemoveUnavailableRegistration(registration.Name, registration.Id)
					}
					continue
				}
				result.Failed(proxyErr)
				handled = true
				break
			}
		}
		if !handled {
			result.Failed(errors.NotFound(fmt.Sprintf("fns: there is no %s service", payload.service)).WithMeta("scope", "endpoints"))
		}
	}
}

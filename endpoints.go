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
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
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
		result.Failed(errors.Warning("fns: send request to endpoint failed").WithMeta("scope", "system"))
	}
	return
}

type remoteUnitPayload struct {
	ctx          Context
	registration *Registration
	fn           string
	argument     Argument
	result       Result
}

type remoteEndpoint struct {
	workerPool   workers.Workers
	registration *Registration
}

func (endpoint *remoteEndpoint) Request(ctx Context, fn string, argument Argument) (result Result) {
	result = NewResult()
	ok := endpoint.workerPool.Execute("remote", &remoteUnitPayload{
		ctx:          ctx,
		registration: endpoint.registration,
		fn:           fn,
		argument:     argument,
		result:       result,
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
	concurrency       int
	workerMaxIdleTime time.Duration
	barrier           Barrier
	discovery         Discovery
}

func newEndpoints(env Environments, opt serviceEndpointsOptions) (v *serviceEndpoints, err error) {
	workerPool, workerPoolErr := workers.New(newEndpointHandler(env), workers.WithConcurrency(opt.concurrency), workers.WithMaxIdleTime(opt.workerMaxIdleTime))
	if workerPoolErr != nil {
		err = fmt.Errorf("fns: create endpoints failed for unable to create workers, %s", workerPoolErr)
		return
	}
	v = &serviceEndpoints{
		appId:      env.AppId(),
		endpoints:  make(map[string]*localEndpoint),
		discovery:  opt.discovery,
		workerPool: workerPool,
	}
	return
}

type serviceEndpoints struct {
	appId      string
	endpoints  map[string]*localEndpoint
	discovery  Discovery
	workerPool workers.Workers
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
	if s.discovery == nil {
		err = errors.NotFound(fmt.Sprintf("fns: there is no %s service", name)).WithMeta("scope", "endpoints")
		return
	}
	registration, getErr := s.discovery.GetRegistration(name)
	if getErr != nil {
		if getErr.Code() == 404 {
			err = errors.NotFound(fmt.Sprintf("fns: there is no %s service endpoint in discovery", name)).WithMeta("scope", "endpoints").WithCause(getErr)
		} else {
			err = errors.Warning(fmt.Sprintf("fns: get %s service endpoint from discovery failed", name)).WithMeta("scope", "endpoints").WithCause(getErr)
		}
		return
	}
	endpoint = &remoteEndpoint{
		workerPool:   s.workerPool,
		registration: registration,
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
	if s.discovery == nil {
		err = errors.NotFound(fmt.Sprintf("fns: there is no %s service", name)).WithMeta("scope", "endpoints")
		return
	}
	registration, getErr := s.discovery.GetExactRegistration(name, registrationId)
	if getErr != nil {
		if getErr.Code() == 404 {
			err = errors.NotFound(fmt.Sprintf("fns: there is no %s service endpoint in discovery", name)).WithMeta("scope", "endpoints").WithCause(getErr)
		} else {
			err = errors.Warning(fmt.Sprintf("fns: get %s service endpoint from discovery failed", name)).WithMeta("scope", "endpoints").WithCause(getErr)
		}
		return
	}
	endpoint = &remoteEndpoint{
		workerPool:   s.workerPool,
		registration: registration,
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

func newEndpointHandler(env Environments) *endpointHandler {
	return &endpointHandler{
		log: env.Log().With("fns", "endpoint"),
	}
}

type endpointHandler struct {
	log logs.Logger
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
	v, err := service.Handle(ctx, fn, arg)
	// span.End()
	span.Finish()
	if err == nil {
		result.Succeed(v)
	} else {
		codeErr, ok := err.(errors.CodeError)
		if ok {
			result.Failed(codeErr)
		} else {
			result.Failed(errors.ServiceError("fns: service handle request failed").WithCause(err))
		}
	}
}

func (h *endpointHandler) handleRemoteAction(payload *remoteUnitPayload) {
	registration := payload.registration
	fn := payload.fn
	parentCtx := payload.ctx.(*context)
	ctx := &context{
		Context:           parentCtx.Context,
		request:           parentCtx.request,
		data:              parentCtx.data,
		log:               parentCtx.runtime.Log().With("service", registration.Name).With("fn", fn).With("registration", fmt.Sprintf("%s:%s:%s", registration.Name, registration.Id, registration.Address)),
		tracer:            parentCtx.tracer,
		serviceComponents: nil,
		runtime:           parentCtx.runtime,
	}
	arg := payload.argument
	result := payload.result
	v, err := registration.proxy().Request(ctx, fn, arg)
	if err == nil {
		result.Succeed(v)
	} else {
		codeErr, ok := err.(errors.CodeError)
		if ok {
			result.Failed(codeErr)
		} else {
			result.Failed(errors.ServiceError("fns: service handle request failed").WithCause(err))
		}
	}
}

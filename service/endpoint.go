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

package service

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/listeners"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"time"
)

type Endpoint interface {
	Request(ctx context.Context, fn string, argument Argument) (result Result)
}

type EndpointDiscovery interface {
	Get(ctx context.Context, service string) (endpoint Endpoint, has bool)
	GetExact(ctx context.Context, service string, id string) (endpoint Endpoint, has bool)
}

// +-------------------------------------------------------------------------------------------------------------------+

type Handler interface {
	Handle(ctx context.Context, r Request) (v interface{}, err errors.CodeError)
}

type Endpoints interface {
	Handler
	Mount(svc Service)
	RegisterInboundChannels(name string, channels listeners.InboundChannels)
	Documents() (v map[string]Document)
	SetupContext(ctx context.Context) context.Context
	Close()
}

type EndpointsOptions struct {
	AppId                 string
	Log                   logs.Logger
	MaxWorkers            int
	MaxIdleWorkerDuration time.Duration
	HandleTimeout         time.Duration
	Barrier               Barrier
	Discovery             EndpointDiscovery
}

func NewEndpoints(options EndpointsOptions) (v Endpoints) {
	maxWorkers := options.MaxWorkers
	if maxWorkers < 1 {
		maxWorkers = 256 * 1024
	}
	maxIdleWorkerDuration := options.MaxIdleWorkerDuration
	if maxIdleWorkerDuration < 1 {
		maxIdleWorkerDuration = 3 * time.Second
	}
	ws := workers.New(workers.MaxWorkers(maxWorkers), workers.MaxIdleWorkerDuration(maxIdleWorkerDuration))
	handleTimeout := options.HandleTimeout
	if handleTimeout < 1 {
		handleTimeout = 10 * time.Second
	}
	barrier := options.Barrier
	if barrier == nil {
		barrier = defaultBarrier()
	}
	v = &endpoints{
		appId:   options.AppId,
		log:     options.Log,
		ws:      ws,
		barrier: barrier,
		group: &group{
			appId:     options.AppId,
			log:       options.Log.With("fns", "service group"),
			ws:        ws,
			services:  make(map[string]Service),
			discovery: options.Discovery,
		},
		inboundChannels: make(map[string]listeners.InboundChannels),
		handleTimeout:   handleTimeout,
	}
	return
}

type endpoints struct {
	appId           string
	log             logs.Logger
	ws              workers.Workers
	barrier         Barrier
	group           *group
	inboundChannels map[string]listeners.InboundChannels
	handleTimeout   time.Duration
}

func (e *endpoints) Handle(ctx context.Context, r Request) (v interface{}, err errors.CodeError) {
	service, fn := r.Fn()
	barrierKey := fmt.Sprintf("%s:%s:%s", service, fn, r.Hash())
	var cancel func()
	ctx, cancel = context.WithTimeout(ctx, e.handleTimeout)
	handleResult, handleErr, _ := e.barrier.Do(ctx, barrierKey, func() (v interface{}, err errors.CodeError) {
		ctx = e.SetupContext(ctx)
		ctx = SetRequest(ctx, r)
		ep, has := e.group.Get(ctx, service)
		if !has {
			err = errors.NotFound("fns: service was not found").WithMeta("service", service)
			return
		}
		ctx = SetTracer(ctx)
		result := ep.Request(ctx, fn, r.Argument())
		resultValue, hasResult, handleErr := result.Value(ctx)
		if handleErr != nil {
			err = handleErr
		} else {
			if hasResult {
				v = resultValue
			}
		}
		tryReportTracer(ctx)
		return
	})
	e.barrier.Forget(ctx, barrierKey)
	cancel()
	if handleErr != nil {
		err = handleErr.WithMeta("requestId", r.Id())
		return
	}
	if handleResult != nil {
		v = handleResult
	}
	return
}

func (e *endpoints) Mount(svc Service) {
	e.group.add(svc)
}

func (e *endpoints) RegisterInboundChannels(name string, channels listeners.InboundChannels) {
	e.inboundChannels[name] = channels
	return
}

func (e *endpoints) SetupContext(ctx context.Context) context.Context {
	if getRuntime(ctx) == nil {
		ctx = initContext(ctx, e.appId, e.log, e.ws, e.group, e.inboundChannels)
	}
	return ctx
}

func (e *endpoints) Documents() (v map[string]Document) {
	v = e.group.documents()
	return
}

func (e *endpoints) Close() {
	e.ws.Close()
	e.group.close()
}

// +-------------------------------------------------------------------------------------------------------------------+

func newEndpoint(ws workers.Workers, svc Service) *endpoint {
	return &endpoint{ws: ws, svc: svc}
}

type endpoint struct {
	ws  workers.Workers
	svc Service
}

func (e *endpoint) Request(ctx context.Context, fn string, argument Argument) (result Result) {
	fr := NewResult()
	_, hasRequest := GetRequest(ctx)
	if !hasRequest {
		req, reqErr := NewInternalRequest(e.svc.Name(), fn, argument)
		if reqErr != nil {
			fr.Failed(errors.Warning("fns: there is no request in context, then to create internal request but failed").WithCause(reqErr).WithMeta("service", e.svc.Name()).WithMeta("fn", fn))
			result = fr
			return
		}
		ctx = SetRequest(ctx, req)
	}
	if !e.ws.Dispatch(newFn(ctx, e.svc, fn, argument, fr)) {
		fr.Failed(errors.Unavailable("fns: service is overload").WithMeta("fns", "overload").WithMeta("service", e.svc.Name()).WithMeta("fn", fn))
	}
	result = fr
	return
}

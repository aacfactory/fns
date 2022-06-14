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
	"github.com/aacfactory/json"
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

type Endpoints interface {
	Handle(ctx context.Context, r Request) (v []byte, err errors.CodeError)
	Mount(svc Service)
	Documents() (v map[string]Document)
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
		log:     options.Log,
		ws:      ws,
		barrier: barrier,
		group: &group{
			appId:     options.AppId,
			log:       options.Log.With("fns", "services"),
			ws:        ws,
			services:  make(map[string]Service),
			discovery: options.Discovery,
		},
		handleTimeout: handleTimeout,
	}
	return
}

type endpoints struct {
	log           logs.Logger
	ws            workers.Workers
	barrier       Barrier
	group         *group
	handleTimeout time.Duration
}

func (e *endpoints) Handle(ctx context.Context, r Request) (v []byte, err errors.CodeError) {
	service, fn := r.Fn()
	barrierKey := fmt.Sprintf("%s:%s:%s", service, fn, r.Hash())
	var cancel func()
	ctx, cancel = context.WithTimeout(ctx, e.handleTimeout)
	handleResult, handleErr, _ := e.barrier.Do(ctx, barrierKey, func() (v interface{}, err errors.CodeError) {
		ctx = initContext(ctx, e.log, e.ws, e.group)
		ctx = SetRequest(ctx, r)
		ep, has := e.group.Get(ctx, service)
		if !has {
			err = errors.NotFound("fns: service was not found").WithMeta("service", service)
			return
		}
		ctx = SetTracer(ctx)
		result := ep.Request(ctx, fn, r.Argument())
		p := json.RawMessage{}
		hasResult, handleErr := result.Get(ctx, &p)
		if handleErr != nil {
			err = handleErr
		} else {
			if hasResult {
				v = p
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
		v = handleResult.(json.RawMessage)
	}
	return
}

func (e *endpoints) Mount(svc Service) {
	e.group.add(svc)
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
	if !e.ws.Dispatch(newFn(ctx, e.svc, fn, argument, fr)) {
		fr.Failed(errors.Unavailable("fns: service is overload").WithMeta("fns", "overload"))
	}
	result = fr
	return
}

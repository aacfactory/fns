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
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"os"
	"sync/atomic"
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
	Services() (services []string)
	Listen() (err error)
	Documents() (v map[string]Document)
	SetupContext(ctx context.Context) context.Context
	Close()
}

type EndpointsOptions struct {
	AppId                 string
	AppStopChan           chan os.Signal
	Running               *commons.SafeFlag
	Log                   logs.Logger
	MaxWorkers            int
	MaxIdleWorkerDuration time.Duration
	HandleTimeout         time.Duration
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
	v = &endpoints{
		appId:       options.AppId,
		appStopChan: options.AppStopChan,
		running:     options.Running,
		log:         options.Log,
		ws:          ws,
		group: &group{
			appId:     options.AppId,
			log:       options.Log.With("fns", "service group"),
			ws:        ws,
			services:  make(map[string]Service),
			discovery: options.Discovery,
		},
		handleTimeout: handleTimeout,
	}
	return
}

type endpoints struct {
	appId         string
	appStopChan   chan os.Signal
	running       *commons.SafeFlag
	log           logs.Logger
	ws            workers.Workers
	group         *group
	handleTimeout time.Duration
}

func (e *endpoints) Handle(ctx context.Context, r Request) (v interface{}, err errors.CodeError) {
	service, fn := r.Fn()
	var cancel func()
	ctx, cancel = context.WithTimeout(ctx, e.handleTimeout)
	ctx = e.SetupContext(ctx)
	ctx = SetRequest(ctx, r)
	ep, has := e.group.Get(ctx, service)
	if !has {
		cancel()
		err = errors.NotFound("fns: service was not found").WithMeta("service", service).WithMeta("requestId", r.Id())
		return
	}
	ctx = SetTracer(ctx)
	result := ep.Request(ctx, fn, r.Argument())
	resultValue, hasResultValue, handleErr := result.Value(ctx)
	if handleErr != nil {
		err = handleErr.WithMeta("requestId", r.Id())
	} else {
		if hasResultValue {
			v = resultValue
		} else {
			v = &Empty{}
		}
	}
	tryReportTracer(ctx)
	cancel()
	return
}

func (e *endpoints) Mount(svc Service) {
	e.group.add(svc)
}

func (e *endpoints) Services() (services []string) {
	services = make([]string, 0, 1)
	for _, service := range e.group.services {
		services = append(services, service.Name())
	}
	return
}

func (e *endpoints) Listen() (err error) {
	errCh := make(chan error, 8)
	lns := 0
	closed := int64(0)
	for _, svc := range e.group.services {
		ln, ok := svc.(Listenable)
		if !ok {
			continue
		}
		lns++
		ctx := e.SetupContext(context.TODO())
		go func(ctx context.Context, ln Listenable, errCh chan error) {
			lnErr := ln.Listen(ctx)
			if lnErr != nil {
				lnErr = errors.Warning(fmt.Sprintf("fns: %s listen falied", ln.Name())).WithCause(lnErr).WithMeta("service", ln.Name())
				if atomic.LoadInt64(&closed) == 0 {
					errCh <- lnErr
				}
			}
		}(ctx, ln, errCh)
	}
	if lns == 0 {
		close(errCh)
		return
	}
	select {
	case lnErr := <-errCh:
		atomic.AddInt64(&closed, 1)
		err = errors.Warning("fns: endpoints listen failed").WithCause(lnErr)
		break
	case <-time.After(time.Duration(lns*3) * time.Second):
		break
	}
	close(errCh)
	return
}

func (e *endpoints) SetupContext(ctx context.Context) context.Context {
	if getRuntime(ctx) == nil {
		ctx = initContext(ctx, e.appId, e.appStopChan, e.running, e.log, e.ws, e.group)
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

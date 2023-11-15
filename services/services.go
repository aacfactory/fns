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

package services

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/log"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

func New(id string, version versions.Version, log logs.Logger, config Config, worker workers.Workers) EndpointsManager {
	return &Services{
		log:     log.With("fns", "services"),
		config:  config,
		id:      id,
		version: version,
		values:  make(Deployed, 0, 1),
		worker:  worker,
	}
}

type Services struct {
	log     logs.Logger
	config  Config
	id      string
	version versions.Version
	values  Deployed
	worker  workers.Workers
}

func (s *Services) Add(service Service) (err error) {
	name := strings.TrimSpace(service.Name())
	if _, has := s.values.Find([]byte(name)); has {
		err = errors.Warning("fns: services add service failed").WithMeta("service", name).WithCause(fmt.Errorf("service has added"))
		return
	}
	config, configErr := s.config.Get(name)
	if configErr != nil {
		err = errors.Warning("fns: services add service failed").WithMeta("service", name).WithCause(configErr)
		return
	}
	constructErr := service.Construct(Options{
		Id:      s.id,
		Version: s.version,
		Log:     s.log.With("service", name),
		Config:  config,
	})
	if constructErr != nil {
		err = errors.Warning("fns: services add service failed").WithMeta("service", name).WithCause(constructErr)
		return
	}
	s.values = s.values.Add(service)
	return
}

func (s *Services) Info() (infos EndpointInfos) {
	infos = make(EndpointInfos, 0, s.values.Len())
	for _, value := range s.values {
		internal := value.Internal()
		functions := make(FnInfos, 0, len(value.Functions()))
		for _, fn := range value.Functions() {
			functions = append(functions, FnInfo{
				Name:     fn.Name(),
				Readonly: fn.Readonly(),
				Internal: internal || fn.Internal(),
			})
		}
		infos = append(infos, EndpointInfo{
			Id:        s.id,
			Name:      value.Name(),
			Version:   s.version,
			Internal:  internal,
			Functions: functions,
			Document:  value.Document(),
		})
	}
	sort.Sort(infos)
	return
}

func (s *Services) Get(_ context.Context, name []byte, options ...EndpointGetOption) (endpoint Endpoint, has bool) {
	if len(options) > 0 {
		opt := EndpointGetOptions{
			id:              nil,
			requestVersions: nil,
		}
		for _, option := range options {
			option(&opt)
		}
		if len(opt.id) > 0 {
			if s.id != string(opt.id) {
				return
			}
		}
		if len(opt.requestVersions) > 0 {
			if !opt.requestVersions.Accept(name, s.version) {
				return
			}
		}
	}
	endpoint, has = s.values.Find(name)
	return
}

func (s *Services) Request(ctx context.Context, name []byte, fn []byte, param interface{}, options ...RequestOption) (response Response, err error) {
	// valid params
	if len(name) == 0 {
		err = errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("name is nil"))
		return
	}
	if len(fn) == 0 {
		err = errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("fn is nil"))
		return
	}

	// request
	req := AcquireRequest(ctx, name, fn, param, options...)

	var endpointGetOptions []EndpointGetOption
	if endpointId := req.Header().EndpointId(); len(endpointId) > 0 {
		endpointGetOptions = make([]EndpointGetOption, 0, 1)
		endpointGetOptions = append(endpointGetOptions, EndpointId(endpointId))
	}
	if acceptedVersions := req.Header().AcceptedVersions(); len(acceptedVersions) > 0 {
		if endpointGetOptions == nil {
			endpointGetOptions = make([]EndpointGetOption, 0, 1)
		}
		endpointGetOptions = append(endpointGetOptions, EndpointVersions(acceptedVersions))
	}
	endpoint, found := s.Get(ctx, name, endpointGetOptions...)
	if !found {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("service", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		ReleaseRequest(req)
		return
	}

	function, hasFunction := endpoint.Functions().Find(fn)
	if !hasFunction {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("service", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		ReleaseRequest(req)
		return
	}

	// ctx >>>
	ctx = req
	// log
	log.With(ctx, s.log.With("service", bytex.ToString(name)).With("fn", bytex.ToString(fn)))
	// components
	service, ok := endpoint.(Service)
	if ok {
		components := service.Components()
		if len(components) > 0 {
			WithComponents(ctx, components)
		}
	}
	// ctx <<<
	// tracing
	trace, hasTrace := tracings.Load(ctx)
	if hasTrace {
		trace.Begin(req.Header().ProcessId(), name, fn, "scope", "local")
	}
	// promise
	promise, future := futures.New()
	// dispatch
	dispatched := s.worker.Dispatch(ctx, fnTask{
		fn:      function,
		promise: promise,
	})
	if !dispatched {
		// tracing
		if hasTrace {
			trace.Finish("succeed", "false", "cause", "***TOO MANY REQUEST***")
		}
		promise.Failed(
			errors.New(http.StatusTooManyRequests, "***TOO MANY REQUEST***", "fns: too may request, try again later.").
				WithMeta("service", bytex.ToString(name)).
				WithMeta("fn", bytex.ToString(fn)),
		)
	}
	response, err = future.Get(ctx)
	ReleaseRequest(req)
	return
}

func (s *Services) Listen(ctx context.Context) (err error) {
	errs := errors.MakeErrors()
	for _, endpoint := range s.values {
		ln, ok := endpoint.(Listenable)
		if !ok {
			continue
		}
		errCh := make(chan error, 1)

		lnCtx := context.WithValue(ctx, bytex.FromString("listener"), ln.Name())
		log.With(lnCtx, s.log.With("service", ln.Name()))
		if components := ln.Components(); len(components) > 0 {
			WithComponents(lnCtx, components)
		}
		go func(ctx context.Context, ln Listenable, errCh chan error) {
			lnErr := ln.Listen(ctx)
			if lnErr != nil {
				errCh <- lnErr
			}
		}(lnCtx, ln, errCh)
		select {
		case lnErr := <-errCh:
			errs.Append(lnErr)
			break
		case <-time.After(5 * time.Second):
			break
		}
		close(errCh)
		if s.log.DebugEnabled() {
			s.log.Debug().With("service", endpoint.Name()).Message("fns: service is listening...")
		}
	}
	if len(errs) > 0 {
		err = errs.Error()
	}
	return
}

func (s *Services) Shutdown(ctx context.Context) {
	wg := new(sync.WaitGroup)
	ch := make(chan struct{}, 1)
	for _, endpoint := range s.values {
		wg.Add(1)
		go func(ctx context.Context, endpoint Endpoint, wg *sync.WaitGroup) {
			endpoint.Shutdown(ctx)
			wg.Done()
		}(ctx, endpoint, wg)
	}
	go func(ch chan struct{}, wg *sync.WaitGroup) {
		wg.Wait()
		ch <- struct{}{}
		close(ch)
	}(ch, wg)
	select {
	case <-ctx.Done():
		break
	case <-ch:
		break
	}
}

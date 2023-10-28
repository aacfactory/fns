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
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services/metrics"
	"github.com/aacfactory/fns/services/tracing"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"golang.org/x/sync/singleflight"
	"strconv"
	"strings"
	"time"
)

func New(id []byte, version versions.Version, log logs.Logger, config Config, worker workers.Workers, discovery Discovery) *Services {
	return &Services{
		log:       log.With("fns", "services"),
		config:    config,
		id:        id,
		version:   version,
		values:    make(map[string]Endpoint),
		listeners: make([]Listenable, 0, 1),
		discovery: discovery,
		group:     new(singleflight.Group),
		worker:    worker,
	}
}

type Services struct {
	log       logs.Logger
	config    Config
	id        []byte
	version   versions.Version
	values    map[string]Endpoint
	listeners []Listenable
	discovery Discovery
	group     *singleflight.Group
	worker    workers.Workers
}

func (s *Services) Add(service Service) (err error) {
	name := strings.TrimSpace(service.Name())
	if _, has := s.values[name]; has {
		err = errors.Warning("fns: services add service failed").WithMeta("service", name).WithCause(fmt.Errorf("service has added"))
		return
	}
	config, configErr := s.config.Get(name)
	if configErr != nil {
		err = errors.Warning("fns: services add service failed").WithMeta("service", name).WithCause(configErr)
		return
	}
	constructErr := service.Construct(Options{
		Log:    s.log.With("service", name),
		Config: config,
	})
	if constructErr != nil {
		err = errors.Warning("fns: services add service failed").WithMeta("service", name).WithCause(constructErr)
		return
	}
	s.values[name] = service
	ln, ok := service.(Listenable)
	if ok {
		s.listeners = append(s.listeners, ln)
	}
	return
}

func (s *Services) Request(ctx context.Context, name []byte, fn []byte, arg Argument, options ...RequestOption) futures.Future {
	// promise
	promise, future := futures.New()
	// valid params
	if len(name) == 0 {
		promise.Failed(errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("name is nil")))
		return future

	}
	if len(fn) == 0 {
		promise.Failed(errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("fn is nil")))
		return future
	}
	if arg == nil {
		arg = NewArgument(nil)
	}

	// request
	req := NewRequest(ctx, name, fn, arg, options...)

	// endpoint
	var endpoint Endpoint
	remoted := false
	if len(req.Header().EndpointId()) == 0 {
		local, inLocal := s.values[bytex.ToString(name)]
		if inLocal {
			// accept versions
			accepted := req.Header().AcceptedVersions().Accept(name, s.version)
			if !accepted {
				promise.Failed(
					errors.NotFound("fns: endpoint was not found").
						WithCause(fmt.Errorf("version was not match")).
						WithMeta("service", bytex.ToString(name)).
						WithMeta("fn", bytex.ToString(fn)),
				)
				return future
			}
			endpoint = local
		} else {
			if s.discovery == nil {
				promise.Failed(
					errors.NotFound("fns: endpoint was not found").
						WithMeta("service", bytex.ToString(name)).
						WithMeta("fn", bytex.ToString(fn)),
				)
				return future
			}
			remote, fetched := s.discovery.Get(ctx, name, EndpointVersions(req.Header().AcceptedVersions()))
			if !fetched {
				promise.Failed(
					errors.NotFound("fns: endpoint was not found").
						WithMeta("service", bytex.ToString(name)).
						WithMeta("fn", bytex.ToString(fn)),
				)
				return future
			}
			endpoint = remote
			remoted = true
		}
	} else {
		if bytex.Equal(s.id, req.Header().EndpointId()) {
			local, inLocal := s.values[bytex.ToString(name)]
			if inLocal {
				// accept versions
				accepted := req.Header().AcceptedVersions().Accept(name, s.version)
				if !accepted {
					promise.Failed(
						errors.NotFound("fns: endpoint was not found").
							WithCause(fmt.Errorf("version was not match")).
							WithMeta("service", bytex.ToString(name)).
							WithMeta("fn", bytex.ToString(fn)).
							WithMeta("endpointId", bytex.ToString(req.Header().EndpointId())),
					)
					return future
				}
				endpoint = local
			} else {
				promise.Failed(
					errors.NotFound("fns: endpoint was not found").
						WithMeta("service", bytex.ToString(name)).
						WithMeta("fn", bytex.ToString(fn)).
						WithMeta("endpointId", bytex.ToString(req.Header().EndpointId())),
				)
				return future
			}
		} else {
			if s.discovery == nil {
				promise.Failed(
					errors.NotFound("fns: endpoint was not found").
						WithMeta("service", bytex.ToString(name)).
						WithMeta("fn", bytex.ToString(fn)).
						WithMeta("endpointId", bytex.ToString(req.Header().EndpointId())),
				)
				return future
			}
			remote, fetched := s.discovery.Get(ctx, name, EndpointVersions(req.Header().AcceptedVersions()), EndpointId(req.Header().EndpointId()))
			if !fetched {
				promise.Failed(
					errors.NotFound("fns: endpoint was not found").
						WithMeta("service", bytex.ToString(name)).
						WithMeta("fn", bytex.ToString(fn)).
						WithMeta("endpointId", bytex.ToString(req.Header().EndpointId())),
				)
				return future
			}
			endpoint = remote
			remoted = true
		}
	}

	// tracer begin
	var traceEndpoint Endpoint
	if len(req.Header().RequestId()) > 0 {
		traceEndpoint = s.traceEndpoint(ctx)
		if traceEndpoint != nil {
			ctx = tracing.Begin(
				ctx,
				req.Header().RequestId(), req.Header().ProcessId(),
				name, fn,
				"internal", strconv.FormatBool(req.Header().Internal()),
				"hostId", bytex.ToString(s.id),
				"remoted", strconv.FormatBool(remoted),
			)
		}
	}

	// metric begin
	metricEndpoint := s.metricEndpoint(ctx)
	if metricEndpoint != nil {
		ctx = metrics.Begin(ctx, name, fn, req.Header().DeviceId(), req.Header().DeviceIp(), remoted)
	}

	// dispatch
	dispatched := s.worker.Dispatch(ctx, fnTask{
		log:            s.log.With("service", bytex.ToString(name)).With("fn", bytex.ToString(fn)),
		group:          s.group,
		worker:         s.worker,
		traceEndpoint:  traceEndpoint,
		metricEndpoint: metricEndpoint,
		endpoint:       endpoint,
		request:        req,
		promise:        promise,
	})
	if !dispatched {
		promise.Failed(
			ErrServiceOverload.
				WithMeta("service", bytex.ToString(name)).
				WithMeta("fn", bytex.ToString(fn)),
		)
		return future
	}

	return future
}

func (s *Services) Listen(ctx context.Context) (err error) {
	errs := errors.MakeErrors()
	for _, ln := range s.listeners {
		errCh := make(chan error, 1)
		go func(ctx context.Context, ln Listenable, errCh chan error) {
			lnErr := ln.Listen(ctx)
			if lnErr != nil {
				errCh <- lnErr
			}
		}(context.WithValue(ctx, "listener", ln.Name()), ln, errCh)
		select {
		case lnErr := <-errCh:
			errs.Append(lnErr)
			break
		case <-time.After(5 * time.Second):
			break
		}
		close(errCh)
	}
	if len(errs) > 0 {
		err = errs.Error()
	}
	return
}

func (s *Services) traceEndpoint(ctx context.Context) Endpoint {
	local, has := s.values[tracing.ServiceName]
	if has {
		return local
	}
	if s.discovery == nil {
		return nil
	}
	remote, fetched := s.discovery.Get(ctx, bytex.FromString(tracing.ServiceName))
	if fetched {
		return remote
	}
	return nil
}

func (s *Services) metricEndpoint(ctx context.Context) Endpoint {
	local, has := s.values[metrics.ServiceName]
	if has {
		return local
	}
	if s.discovery == nil {
		return nil
	}
	remote, fetched := s.discovery.Get(ctx, bytex.FromString(metrics.ServiceName))
	if fetched {
		return remote
	}
	return nil
}

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
	"bytes"
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/services/metrics"
	"github.com/aacfactory/fns/services/tracing"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"golang.org/x/sync/singleflight"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func New(id []byte, name []byte, version versions.Version, log logs.Logger, config Config, worker workers.Workers, discovery Discovery) *Services {
	return &Services{
		log:       log.With("fns", "services"),
		config:    config,
		id:        id,
		version:   version,
		doc:       documents.NewDocuments(id, name, version),
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
	doc       *documents.Documents
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
	doc := service.Document()
	if doc != nil && !service.Internal() {
		s.doc.Add(doc)
	}
	return
}

func (s *Services) Documents() (v *documents.Documents) {
	v = s.doc
	return
}

func (s *Services) Request(ctx sc.Context, name []byte, fn []byte, arg Argument, options ...RequestOption) (response Response, err error) {
	// valid params
	if len(name) == 0 {
		err = errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("name is nil"))
		return
	}
	if len(fn) == 0 {
		err = errors.Warning("fns: endpoints handle request failed").WithCause(fmt.Errorf("fn is nil"))
		return
	}
	if arg == nil {
		arg = NewArgument(nil)
	}

	// request
	req := AcquireRequest(ctx, name, fn, arg, options...)

	// endpoint
	var endpoint Endpoint
	remoted := false
	if len(req.Header().EndpointId()) == 0 {
		local, inLocal := s.values[bytex.ToString(name)]
		if inLocal {
			// accept versions
			accepted := req.Header().AcceptedVersions().Accept(name, s.version)
			if !accepted {
				err = errors.NotFound("fns: endpoint was not found").
					WithCause(fmt.Errorf("version was not match")).
					WithMeta("service", bytex.ToString(name)).
					WithMeta("fn", bytex.ToString(fn))
				ReleaseRequest(req)
				return
			}
			endpoint = local
		} else {
			if req.Header().Internal() && s.discovery != nil {
				remote, fetched := s.discovery.Get(ctx, name, EndpointVersions(req.Header().AcceptedVersions()))
				if fetched {
					endpoint = remote
					remoted = true
				}
			}
		}
	} else {
		if bytes.Equal(s.id, req.Header().EndpointId()) {
			local, inLocal := s.values[bytex.ToString(name)]
			if inLocal {
				// accept versions
				accepted := req.Header().AcceptedVersions().Accept(name, s.version)
				if !accepted {
					err = errors.NotFound("fns: endpoint was not found").
						WithCause(fmt.Errorf("version was not match")).
						WithMeta("service", bytex.ToString(name)).
						WithMeta("fn", bytex.ToString(fn)).
						WithMeta("endpointId", bytex.ToString(req.Header().EndpointId()))
					ReleaseRequest(req)
					return
				}
				endpoint = local
			}
		} else {
			if req.Header().Internal() && s.discovery != nil {
				remote, fetched := s.discovery.Get(ctx, name, EndpointVersions(req.Header().AcceptedVersions()), EndpointId(req.Header().EndpointId()))
				if fetched {
					endpoint = remote
					remoted = true
				}
			}
		}
	}

	if endpoint == nil {
		err = errors.NotFound("fns: endpoint was not found").
			WithMeta("service", bytex.ToString(name)).
			WithMeta("fn", bytex.ToString(fn))
		ReleaseRequest(req)
		return
	}

	// ctx
	ctx = req

	// tracer begin
	traceEndpoint := s.traceEndpoint(ctx)
	if traceEndpoint != nil && len(req.Header().RequestId()) > 0 {
		ctx = tracing.Begin(
			ctx,
			req.Header().RequestId(), req.Header().ProcessId(),
			name, fn,
			"internal", strconv.FormatBool(req.Header().Internal()),
			"hostId", bytex.ToString(s.id),
			"remoted", strconv.FormatBool(remoted),
		)
	}

	// metric begin
	metricEndpoint := s.metricEndpoint(ctx)
	if metricEndpoint != nil {
		ctx = metrics.Begin(ctx, name, fn, req.Header().DeviceId(), req.Header().DeviceIp(), remoted)
	}

	// promise
	promise, future := futures.New()
	// dispatch
	dispatched := s.worker.Dispatch(ctx, fnTask{
		log:            s.log.With("service", bytex.ToString(name)).With("fn", bytex.ToString(fn)),
		group:          s.group,
		worker:         s.worker,
		traceEndpoint:  traceEndpoint,
		metricEndpoint: metricEndpoint,
		endpoint:       endpoint,
		promise:        promise,
	})
	if !dispatched {
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

func (s *Services) Listen(ctx sc.Context) (err error) {
	errs := errors.MakeErrors()
	for _, ln := range s.listeners {
		errCh := make(chan error, 1)
		lnCtx := sc.WithValue(ctx, bytex.FromString("listener"), ln.Name())
		if components := ln.Components(); len(components) > 0 {
			lnCtx = WithComponents(lnCtx, components)
		}
		go func(ctx sc.Context, ln Listenable, errCh chan error) {
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
	}
	if len(errs) > 0 {
		err = errs.Error()
	}
	return
}

func (s *Services) traceEndpoint(ctx sc.Context) Endpoint {
	local, has := s.values[tracing.EndpointName]
	if has {
		return local
	}
	if s.discovery == nil {
		return nil
	}
	remote, fetched := s.discovery.Get(ctx, bytex.FromString(tracing.EndpointName))
	if fetched {
		return remote
	}
	return nil
}

func (s *Services) metricEndpoint(ctx sc.Context) Endpoint {
	local, has := s.values[metrics.EndpointName]
	if has {
		return local
	}
	if s.discovery == nil {
		return nil
	}
	remote, fetched := s.discovery.Get(ctx, bytex.FromString(metrics.EndpointName))
	if fetched {
		return remote
	}
	return nil
}

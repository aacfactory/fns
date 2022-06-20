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
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
)

type group struct {
	appId     string
	log       logs.Logger
	ws        workers.Workers
	services  map[string]Service
	discovery EndpointDiscovery
}

func (g *group) Get(ctx context.Context, service string) (endpoint Endpoint, has bool) {
	svc, exist := g.services[service]
	if !exist {
		if g.discovery == nil {
			return
		}
		if !CanAccessInternal(ctx) {
			return
		}
		endpoint, has = g.discovery.Get(ctx, service)
		return
	}
	if svc.Internal() {
		if CanAccessInternal(ctx) {
			endpoint = newEndpoint(g.ws, svc)
			has = true
		}
		return
	}
	endpoint = newEndpoint(g.ws, svc)
	has = true
	return
}

func (g *group) GetExact(ctx context.Context, service string, id string) (endpoint Endpoint, has bool) {
	if id == g.appId {
		svc, exist := g.services[service]
		if !exist {
			return
		}
		if svc.Internal() {
			if CanAccessInternal(ctx) {
				endpoint = newEndpoint(g.ws, svc)
				has = true
			}
			return
		}
		endpoint = newEndpoint(g.ws, svc)
		has = true
		return
	}
	if g.discovery == nil {
		return
	}
	if !CanAccessInternal(ctx) {
		return
	}
	endpoint, has = g.discovery.Get(ctx, service)
	return
}

func (g *group) add(svc Service) {
	g.services[svc.Name()] = svc
	return
}

func (g *group) documents() (v map[string]Document) {
	v = make(map[string]Document)
	for name, service := range g.services {
		if service.Internal() || service.Document() == nil {
			continue
		}
		v[name] = service.Document()
	}
	return
}

func (g *group) close() {
	for _, service := range g.services {
		service.Close()
	}
}

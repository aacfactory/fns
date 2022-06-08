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

type Group struct {
	appId     string
	log       logs.Logger
	ws        workers.Workers
	services  map[string]Service
	discovery EndpointDiscovery
}

func (group *Group) Get(ctx context.Context, service string) (endpoint Endpoint, has bool) {
	svc, exist := group.services[service]
	if !exist {
		if group.discovery == nil {
			return
		}
		if !CanAccessInternal(ctx) {
			return
		}
		endpoint, has = group.discovery.Get(ctx, service)
		return
	}
	if svc.Internal() {
		if CanAccessInternal(ctx) {
			endpoint = newEndpoint(group.ws, svc)
			has = true
		}
		return
	}
	endpoint = newEndpoint(group.ws, svc)
	has = true
	return
}

func (group *Group) GetExact(ctx context.Context, service string, id string) (endpoint Endpoint, has bool) {
	if id == group.appId {
		svc, exist := group.services[service]
		if !exist {
			return
		}
		if svc.Internal() {
			if CanAccessInternal(ctx) {
				endpoint = newEndpoint(group.ws, svc)
				has = true
			}
			return
		}
		endpoint = newEndpoint(group.ws, svc)
		has = true
		return
	}
	if group.discovery == nil {
		return
	}
	if !CanAccessInternal(ctx) {
		return
	}
	endpoint, has = group.discovery.Get(ctx, service)
	return
}

func (group *Group) add(svc Service) {
	group.services[svc.Name()] = svc
	return
}

func (group *Group) close() {
	for _, service := range group.services {
		service.Close()
	}
}

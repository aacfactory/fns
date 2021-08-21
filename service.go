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
	"context"
	"fmt"
)

type ServiceRequestHandler interface {
	Build(services []Service) (err error)
	Handle(ctx Context, arg Argument) (result interface{}, err CodeError)
}

func newMappedServiceRequestHandler() ServiceRequestHandler {
	return &mappedServiceRequestHandler{
		serviceMap: make(map[string]Service),
	}
}

type mappedServiceRequestHandler struct {
	serviceMap map[string]Service
}

func (h *mappedServiceRequestHandler) Build(services []Service) (err error) {
	if services == nil || len(services) == 0 {
		err = fmt.Errorf("service request handler build failed for empty services")
		return
	}
	for _, service := range services {
		if service == nil {
			continue
		}
		h.serviceMap[service.Namespace()] = service
	}
	return
}

func (h *mappedServiceRequestHandler) Handle(ctx Context, arg Argument) (result interface{}, err CodeError) {
	service, has := h.serviceMap[ctx.Namespace()]
	if !has {
		err = NotFoundError(fmt.Sprintf("%s/%s was not found", ctx.Namespace(), ctx.FnName()))
		return
	}
	result, err = service.Handle(ctx, arg)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Service interface {
	Namespace() (namespace string)
	Build(context context.Context, config Config, log Logs) (err error)
	Handle(context Context, argument Argument) (result interface{}, err CodeError)
	Close(context context.Context) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

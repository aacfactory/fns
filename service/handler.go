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

type HandleOptions struct {
	AppId                 string
	Log                   logs.Logger
	MaxWorkers            int
	MaxIdleWorkerDuration time.Duration
	HandleTimeout         time.Duration
	Barrier               Barrier
	Discovery             EndpointDiscovery
}

func NewHandler(options HandleOptions) (handler *Handler) {
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
	handler = &Handler{
		log:     options.Log,
		ws:      ws,
		barrier: barrier,
		group: &Group{
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

type Handler struct {
	log           logs.Logger
	ws            workers.Workers
	barrier       Barrier
	group         *Group
	handleTimeout time.Duration
}

func (h *Handler) Handle(ctx context.Context, r Request) (v []byte, err errors.CodeError) {
	service, fn := r.Fn()
	barrierKey := fmt.Sprintf("%s:%s:%s", service, fn, r.Hash())
	handleResult, handleErr, _ := h.barrier.Do(ctx, barrierKey, func() (v interface{}, err errors.CodeError) {
		ctx = initContext(ctx, h.log, h.ws, h.group)
		ctx = setRequest(ctx, r)
		ep, has := h.group.Get(ctx, service)
		if !has {
			err = errors.NotFound("fns: service was not found").WithMeta("service", service)
			return
		}
		ctx = setTracer(ctx)
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, h.handleTimeout)
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
		cancel()
		return
	})
	h.barrier.Forget(ctx, barrierKey)
	if handleErr != nil {
		err = handleErr
		return
	}
	if handleResult != nil {
		v = handleResult.([]byte)
	}
	return
}

func (h *Handler) mount(svc Service) {
	h.group.add(svc)
}

func (h *Handler) Close() {
	h.ws.Close()
	h.group.close()
}

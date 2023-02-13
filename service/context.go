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
	"github.com/aacfactory/fns/shared"
	"github.com/aacfactory/logs"
	"os"
	"sync"
	"time"
)

const (
	contextRuntimeKey    = "@fns_runtime"
	contextLogKey        = "@fns_log"
	contextComponentsKey = "@fns_service_components"
)

func GetLog(ctx context.Context) (log logs.Logger) {
	log = ctx.Value(contextLogKey).(logs.Logger)
	return
}

func withLog(ctx context.Context, log logs.Logger) context.Context {
	ctx = context.WithValue(ctx, contextLogKey, log)
	return ctx
}

func GetComponent(ctx context.Context, key string) (v Component, has bool) {
	vv := ctx.Value(contextComponentsKey)
	if vv == nil {
		return
	}
	cm, typed := vv.(map[string]Component)
	if !typed {
		return
	}
	v, has = cm[key]
	return
}

func withComponents(ctx context.Context, cm map[string]Component) context.Context {
	ctx = context.WithValue(ctx, contextComponentsKey, cm)
	return ctx
}

func CanAccessInternal(ctx context.Context) (ok bool) {
	r, hasRequest := GetRequest(ctx)
	if !hasRequest {
		return
	}
	if r.Internal() {
		ok = true
		return
	}
	t, hasTracer := GetTracer(ctx)
	if !hasTracer {
		return
	}
	if t.Span() == nil {
		return
	}
	ok = t.Span().Parent() != nil
	return
}

func GetEndpoint(ctx context.Context, name string) (v Endpoint, has bool) {
	rt := getRuntime(ctx)
	if rt == nil {
		return
	}
	v, has = rt.discovery.Get(ctx, name)
	return
}

func GetExactEndpoint(ctx context.Context, name string, id string) (v Endpoint, has bool) {
	rt := getRuntime(ctx)
	if rt == nil {
		return
	}
	v, has = rt.discovery.Get(ctx, name, Exact(id))
	return
}

func withRuntime(ctx context.Context, appId string, log logs.Logger, worker Workers, discovery EndpointDiscovery, barrier Barrier, sharedLockers shared.Lockers, sharedStore shared.Store, running *commons.SafeFlag) context.Context {
	if getRuntime(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, contextRuntimeKey, &runtimes{
		appId:         appId,
		log:           log,
		worker:        worker,
		discovery:     discovery,
		barrier:       barrier,
		sharedLockers: sharedLockers,
		sharedStore:   sharedStore,
		running:       running,
	})
}

func getRuntime(ctx context.Context) (v *runtimes) {
	rt := ctx.Value(contextRuntimeKey)
	if rt == nil {
		return nil
	}
	v = rt.(*runtimes)
	return
}

func GetApplicationId(ctx context.Context) (appId string) {
	rt := getRuntime(ctx)
	if rt == nil {
		return
	}
	appId = rt.appId
	return
}

func GetBarrier(ctx context.Context) (barrier Barrier) {
	rt := getRuntime(ctx)
	if rt == nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: barrier was not found")))
		return
	}
	barrier = rt.barrier
	return
}

func SharedStore(ctx context.Context) (store shared.Store) {
	rt := getRuntime(ctx)
	if rt == nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: shared store was not found")))
		return
	}
	store = rt.sharedStore
	return
}

func SharedLocker(ctx context.Context, key []byte, timeout time.Duration) (locker sync.Locker, err errors.CodeError) {
	rt := getRuntime(ctx)
	if rt == nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: shared lockers was not found")))
		return
	}
	locker, err = rt.sharedLockers.Get(ctx, key, timeout)
	if err != nil {
		err = errors.ServiceError("fns: get shared locker failed").WithCause(err)
		return
	}
	return
}

func AbortApplication() {
	os.Exit(9)
}

func Todo(ctx context.Context, endpoints *Endpoints) context.Context {
	rt := getRuntime(ctx)
	if rt != nil {
		return ctx
	}
	return withRuntime(ctx, endpoints.appId, endpoints.log, endpoints.worker, endpoints.discovery, endpoints.barrier, endpoints.sharedLockers, endpoints.sharedStore, endpoints.running)
}

func ApplicationRunning(ctx context.Context) (signal <-chan struct{}) {
	rt := getRuntime(ctx)
	if rt == nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: there is no application runtime")))
		return
	}
	ch := make(chan struct{}, 1)
	go func(rt *runtimes, ch chan struct{}) {
		for {
			if ctx.Err() != nil || rt.running.IsOn() {
				ch <- struct{}{}
				close(ch)
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}(rt, ch)
	signal = ch
	return
}

type runtimes struct {
	appId         string
	running       *commons.SafeFlag
	log           logs.Logger
	worker        Workers
	discovery     EndpointDiscovery
	barrier       Barrier
	sharedLockers shared.Lockers
	sharedStore   shared.Store
}

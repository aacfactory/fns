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
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
	"os"
	"syscall"
)

const (
	contextRuntimeKey    = "_fns_"
	contextLogKey        = "_log_"
	contextComponentsKey = "_components_"
)

func GetLog(ctx context.Context) (log logs.Logger) {
	log = ctx.Value(contextLogKey).(logs.Logger)
	return
}

func SetLog(ctx context.Context, log logs.Logger) context.Context {
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

func setComponents(ctx context.Context, cm map[string]Component) context.Context {
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
	v, has = rt.discovery.GetExact(ctx, name, id)
	return
}

func TODO(ctx context.Context, ep Endpoints) context.Context {
	ep0, ok := ep.(*endpoints)
	if !ok {
		panic("fns: todo failed")
	}
	return initContext(ctx, ep0.appId, ep0.appStopChan, ep0.running, ep0.log, ep0.ws, ep0.group)
}

func initContext(ctx context.Context, appId string, appStopChan chan os.Signal, running *commons.SafeFlag, log logs.Logger, ws workers.Workers, discovery EndpointDiscovery) context.Context {
	ctx = context.WithValue(ctx, contextRuntimeKey, &contextValue{
		appId:       appId,
		appStopChan: appStopChan,
		running:     running,
		log:         log,
		ws:          ws,
		discovery:   discovery,
	})
	return ctx
}

func getRuntime(ctx context.Context) (v *contextValue) {
	rt := ctx.Value(contextRuntimeKey)
	if rt == nil {
		return nil
	}
	v = rt.(*contextValue)
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

func ApplicationIsRunning(ctx context.Context) (ok bool) {
	rt := getRuntime(ctx)
	if rt == nil {
		return
	}
	ok = rt.running.IsOn()
	return
}

func AbortApplication(ctx context.Context) {
	rt := getRuntime(ctx)
	if rt == nil {
		return
	}
	rt.appStopChan <- syscall.SIGABRT
}

type contextValue struct {
	appId       string
	appStopChan chan os.Signal
	running     *commons.SafeFlag
	log         logs.Logger
	ws          workers.Workers
	discovery   EndpointDiscovery
}

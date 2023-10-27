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
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/switchs"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
)

const (
	contextRuntimeKey    = "@fns_runtime"
	contextLogKey        = "@fns_log"
	contextComponentsKey = "@fns_service_components"
)

// todo
// 移回 services，然后增加run函数，把ctx的run给它
type Runtime struct {
	appId      string
	appName    string
	appVersion versions.Version
	status     *switchs.Switch
	log        logs.Logger
	worker     workers.Workers
	endpoints  Endpoints
	barrier    Barrier
	shared     shareds.Shared
}

func (rt *Runtime) AppId() string {
	return rt.appId
}

func (rt *Runtime) AppName() string {
	return rt.appName
}

func (rt *Runtime) AppVersion() versions.Version {
	return rt.appVersion
}

func (rt *Runtime) RootLog() logs.Logger {
	return rt.log
}

func (rt *Runtime) Workers() workers.Workers {
	return rt.worker
}

func (rt *Runtime) Discovery() EndpointDiscovery {
	return rt.discovery
}

func (rt *Runtime) Barrier() Barrier {
	return rt.barrier
}

func (rt *Runtime) Shared() shareds.Shared {
	return rt.shared
}

func (rt *Runtime) Signature() signatures.Signature {
	return rt.signer
}

func (rt *Runtime) SetIntoContext(ctx context.Context) context.Context {
	if ctx == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: runtime must be set into a non nil context")))
	}
	return context.WithValue(ctx, contextRuntimeKey, rt)
}

func GetRuntime(ctx context.Context) (v *Runtime) {
	rt := ctx.Value(contextRuntimeKey)
	if rt == nil {
		return nil
	}
	v = rt.(*Runtime)
	return
}

func GetLog(ctx context.Context) (log logs.Logger) {
	log = ctx.Value(contextLogKey).(logs.Logger)
	return
}

func withLog(ctx context.Context, log logs.Logger) context.Context {
	ctx = context.WithValue(ctx, contextLogKey, log)
	return ctx
}

func GetEndpoint(ctx context.Context, name string, options ...EndpointDiscoveryGetOption) (v Endpoint, has bool) {
	rt := GetRuntime(ctx)
	if rt == nil {
		return
	}
	v, has = rt.discovery.Get(ctx, name, options...)
	return
}

/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package runtime

import (
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/commons/switchs"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/workers"
)

func New(id string, name string, version versions.Version, status *switchs.Switch, log logs.Logger, worker workers.Workers, endpoints services.Endpoints, barrier barriers.Barrier, shared shareds.Shared) *Runtime {
	return &Runtime{
		appId:      []byte(id),
		appName:    name,
		appVersion: version,
		status:     status,
		log:        log,
		worker:     worker,
		endpoints:  endpoints,
		barrier:    barrier,
		shared:     shared,
	}
}

type Runtime struct {
	appId      []byte
	appName    string
	appVersion versions.Version
	status     *switchs.Switch
	log        logs.Logger
	worker     workers.Workers
	endpoints  services.Endpoints
	barrier    barriers.Barrier
	shared     shareds.Shared
}

func (rt *Runtime) AppId() []byte {
	return rt.appId
}

func (rt *Runtime) AppName() string {
	return rt.appName
}

func (rt *Runtime) AppVersion() versions.Version {
	return rt.appVersion
}

func (rt *Runtime) Running() (running bool, serving bool) {
	return rt.status.IsOn()
}

func (rt *Runtime) RootLog() logs.Logger {
	return rt.log
}

func (rt *Runtime) Workers() workers.Workers {
	return rt.worker
}

func (rt *Runtime) Endpoints() services.Endpoints {
	return rt.endpoints
}

func (rt *Runtime) Barrier() barriers.Barrier {
	return rt.barrier
}

func (rt *Runtime) Shared() shareds.Shared {
	return rt.shared
}

func (rt *Runtime) TryExecute(ctx context.Context, task workers.Task) bool {
	name := ""
	named, ok := task.(workers.NamedTask)
	if ok {
		name = named.Name()
	} else {
		name = "[task]"
	}
	nc := context.Fork(ctx)
	logs.With(nc, rt.log.With("task", name))
	return rt.worker.Dispatch(nc, task)
}

func (rt *Runtime) Execute(ctx context.Context, task workers.Task) {
	name := ""
	named, ok := task.(workers.NamedTask)
	if ok {
		name = named.Name()
	} else {
		name = "[task]"
	}
	nc := context.Fork(ctx)
	logs.With(nc, rt.log.With("task", name))
	rt.worker.MustDispatch(nc, task)
	return
}

package runtime

import (
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/commons/switchs"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	fLog "github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
)

func New(id string, name string, version versions.Version, status *switchs.Switch, log logs.Logger, worker workers.Workers, endpoints services.Endpoints, barrier barriers.Barrier, shared shareds.Shared) *Runtime {
	return &Runtime{
		appId:      id,
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
	appId      string
	appName    string
	appVersion versions.Version
	status     *switchs.Switch
	log        logs.Logger
	worker     workers.Workers
	endpoints  services.Endpoints
	barrier    barriers.Barrier
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

func (rt *Runtime) Running() (running bool, upped bool) {
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
	name := "[task]"
	named, ok := task.(workers.NamedTask)
	if ok {
		name = named.Name()
	}
	fLog.With(ctx, rt.log.With("task", name))
	With(ctx, rt)
	return rt.worker.Dispatch(ctx, task)
}

func (rt *Runtime) Execute(ctx context.Context, task workers.Task) {
	name := "[task]"
	named, ok := task.(workers.NamedTask)
	if ok {
		name = named.Name()
	}
	fLog.With(ctx, rt.log.With("task", name))
	With(ctx, rt)
	rt.worker.MustDispatch(ctx, task)
	return
}

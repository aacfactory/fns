package runtime

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/switchs"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
)

const (
	contextKey    = "@fns:runtime"
	contextLogKey = "@fns:log"
)

func Get(ctx context.Context) *Runtime {
	v := ctx.Value(contextKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no runtime in context")))
	}
	rt, ok := v.(*Runtime)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: '@fns:runtime' of context is not runtime")))
	}
	return rt
}

func With(ctx context.Context, rt *Runtime) context.Context {
	return context.WithValue(ctx, contextKey, rt)
}

func Log(ctx context.Context) logs.Logger {
	v := ctx.Value(contextLogKey)
	if v == nil {
		rt := Get(ctx)
		return rt.RootLog()
	}
	log, ok := v.(logs.Logger)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: '@fns:log' of context is not runtime")))
	}
	return log
}

func WithLog(ctx context.Context, log logs.Logger) context.Context {
	ctx = context.WithValue(ctx, contextLogKey, log)
	return ctx
}

type Runtime struct {
	appId      string
	appName    string
	appVersion versions.Version
	status     *switchs.Switch
	log        logs.Logger
	worker     workers.Workers
	discovery  services.Discovery
	barrier    barriers.Barrier
	shared     shareds.Shared
	signature  signatures.Signature
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

func (rt *Runtime) Running() bool {
	return rt.status.IsOn()
}

func (rt *Runtime) RootLog() logs.Logger {
	return rt.log
}

func (rt *Runtime) Workers() workers.Workers {
	return rt.worker
}

func (rt *Runtime) Discovery() services.Discovery {
	return rt.discovery
}

func (rt *Runtime) Barrier() barriers.Barrier {
	return rt.barrier
}

func (rt *Runtime) Shared() shareds.Shared {
	return rt.shared
}

func (rt *Runtime) Signature() signatures.Signature {
	return rt.signature
}

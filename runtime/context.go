package runtime

import (
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/workers"
	"time"
)

const (
	contextKey = "@fns:context:runtime"
)

func With(ctx sc.Context, rt *Runtime) sc.Context {
	return context.WithValue(ctx, bytex.FromString(contextKey), rt)
}

func Load(ctx sc.Context) *Runtime {
	v := ctx.Value(contextKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no runtime in context")))
		return nil
	}
	rt, ok := v.(*Runtime)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: runtime in context is not github.com/aacfactory/fns/runtime.Runtime")))
		return nil
	}
	return rt
}

func TryExecute(ctx sc.Context, task workers.Task) bool {
	rt := Load(ctx)
	return rt.TryExecute(ctx, task)
}

func Execute(ctx sc.Context, task workers.Task) {
	rt := Load(ctx)
	rt.Execute(ctx, task)
}

func Barrier(ctx sc.Context, key []byte, fn func() (result interface{}, err error)) (result barriers.Result, err error) {
	rt := Load(ctx)
	barrier := rt.Barrier()
	result, err = barrier.Do(ctx, key, fn)
	barrier.Forget(ctx, key)
	return
}

func AcquireLocker(ctx sc.Context, key []byte, ttl time.Duration) (locker shareds.Locker, err error) {
	rt := Load(ctx)
	locker, err = rt.Shared().Lockers().Acquire(ctx, key, ttl)
	return
}

func SharedStore(ctx sc.Context) (store shareds.Store) {
	rt := Load(ctx)
	store = rt.Shared().Store()
	return
}

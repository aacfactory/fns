package runtime

import (
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
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

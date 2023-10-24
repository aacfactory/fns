package runtime

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
)

const (
	contextKey = "@fns:runtime"
)

func With(ctx context.Context, rt *Runtime) context.Context {
	return context.WithValue(ctx, contextKey, rt)
}

func Load(ctx context.Context) *Runtime {
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

package log

import (
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/logs"
)

const (
	contextKey = "@fns:context:log"
)

func With(ctx sc.Context, v logs.Logger) sc.Context {
	return context.WithValue(ctx, bytex.FromString(contextKey), v)
}

func Load(ctx sc.Context) logs.Logger {
	v := ctx.Value(contextKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no log in context")))
		return nil
	}
	lg, ok := v.(logs.Logger)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: contextKey in context is not github.com/aacfactory/logs.Logger")))
		return nil
	}
	return lg
}

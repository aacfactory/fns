package log

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
)

const (
	ContextKey = "@fns:context:log"
)

func With(ctx context.Context, v logs.Logger) context.Context {
	return context.WithValue(ctx, ContextKey, v)
}

func Load(ctx context.Context) logs.Logger {
	v := ctx.Value(ContextKey)
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

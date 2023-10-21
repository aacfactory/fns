package logger

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
)

const (
	contextKey = "@fns:context:log"
)

func With(ctx context.Context, log logs.Logger) context.Context {
	return context.WithValue(ctx, contextKey, log)
}

func Log(ctx context.Context) logs.Logger {
	v := ctx.Value(contextKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no log in context")))
		return nil
	}
	log, ok := v.(logs.Logger)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: contextKey in context is not github.com/aacfactory/logs.Logger")))
		return nil
	}
	return log
}

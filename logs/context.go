package logs

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/logs"
)

var (
	contextKey = []byte("@fns:context:log")
)

func With(ctx context.Context, v logs.Logger) {
	ctx.SetLocalValue(contextKey, v)
}

func Load(ctx context.Context) logs.Logger {
	v := ctx.LocalValue(contextKey)
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

package tracings

import "github.com/aacfactory/fns/context"

var (
	contextKey = []byte("@fns:context:tracings")
)

func With(ctx context.Context, trace *Trace) {
	ctx.SetLocalValue(contextKey, trace)
}

func Load(ctx context.Context) (trace *Trace, found bool) {
	v := ctx.LocalValue(contextKey)
	if v == nil {
		return
	}
	trace, found = v.(*Trace)
	return
}

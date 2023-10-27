package tracing

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"time"
)

const (
	contextSpanKey = "@fns:context:tracing:span"
)

type Tags map[string]string

func (tags Tags) Merge(ss []string) (ok bool) {
	n := len(ss)
	if n == 0 {
		ok = true
		return
	}
	if n%2 != 0 {
		return
	}
	for i := 0; i < n; i += 2 {
		k := ss[i]
		v := ss[i+1]
		tags[k] = v
	}
	ok = true
	return
}

type Span struct {
	Id       []byte
	Service  []byte
	Fn       []byte
	Beg      time.Time
	End      time.Time
	Waiting  time.Duration
	Handling time.Duration
	Latency  time.Duration
	Tags     Tags
	Children []*Span
	parent   *Span
}

func loadSpan(ctx context.Context) *Span {
	v := ctx.Value(contextSpanKey)
	if v == nil {
		return nil
	}
	sp, ok := v.(*Span)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: load span from context failed, type is not github.com/aacfactory/fns/services/tracing.Span")))
		return nil
	}
	return sp
}

func withSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, contextSpanKey, span)
}

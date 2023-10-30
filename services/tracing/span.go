package tracing

import (
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
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

func (span *Span) mountChildrenParent() {
	for _, child := range span.Children {
		if child.parent == nil {
			child.parent = span
		}
		child.mountChildrenParent()
	}
}

func LoadSpan(ctx sc.Context) *Span {
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

func withSpan(ctx sc.Context, span *Span) sc.Context {
	return context.WithValue(ctx, bytex.FromString(contextSpanKey), span)
}

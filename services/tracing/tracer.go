package tracing

import (
	"bytes"
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"time"
)

const (
	EndpointName = "tracings"
	ReportFnName = "report"
)

const (
	contextTracerIdKey = "@fns:context:tracing:tracerId"
)

type Tracer struct {
	Id   []byte `json:"id"`
	Span *Span  `json:"span"`
}

func Begin(ctx sc.Context, tid []byte, sid []byte, service []byte, fn []byte, tags ...string) sc.Context {
	if len(tid) == 0 || len(sid) == 0 || len(service) == 0 || len(fn) == 0 {
		return ctx
	}
	tagsLen := len(tags)
	if tagsLen > 0 && tagsLen%2 != 0 {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: begin trace failed cause tags is invalid")))
		return ctx
	}

	span := LoadSpan(ctx)
	if span == nil {
		// init
		span = &Span{
			Id:       sid,
			Service:  service,
			Fn:       fn,
			Beg:      time.Now(),
			End:      time.Time{},
			Waiting:  0,
			Handling: 0,
			Latency:  0,
			Tags:     make(Tags),
			Children: make([]*Span, 0, 1),
			parent:   nil,
		}
		span.Tags.Merge(tags)
		ctx = context.WithValue(ctx, bytex.FromString(contextTracerIdKey), tid)
	} else {
		if bytes.Equal(sid, span.Id) {
			return ctx
		}
		child := &Span{
			Id:       sid,
			Service:  service,
			Fn:       fn,
			Beg:      time.Now(),
			End:      time.Time{},
			Waiting:  0,
			Handling: 0,
			Latency:  0,
			Tags:     make(Tags),
			Children: make([]*Span, 0, 1),
			parent:   span,
		}
		child.Tags.Merge(tags)
		if span.End.IsZero() {
			// as current's child
			span.Children = append(span.Children, child)
		} else {
			// as parent's child
			span.parent.Children = append(span.parent.Children, child)
		}
		span = child
	}
	return withSpan(ctx, span)
}

func MarkBeginHandling(ctx sc.Context, tags ...string) {
	span := LoadSpan(ctx)
	if span == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: trace mark begin handling failed cause not begin")))
		return
	}
	tagsLen := len(tags)
	if tagsLen > 0 && tagsLen%2 != 0 {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: trace mark begin handling failed cause tags is invalid")))
		return
	}
	span.Tags.Merge(tags)
	span.Waiting = time.Now().Sub(span.Beg)
	return
}

func MountSpan(ctx sc.Context, span *Span) {
	if span == nil {
		return
	}
	parent := LoadSpan(ctx)
	if parent == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: trace mark begin handling failed cause not begin")))
		return
	}
	span.mountChildrenParent()
	span.parent = parent
	parent.Children = append(parent.Children, span)
}

func Tagging(ctx sc.Context, tags ...string) {
	tagsLen := len(tags)
	if tagsLen == 0 {
		return
	}
	if tagsLen%2 != 0 {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: trace add tags failed cause tags is invalid")))
		return
	}
	span := LoadSpan(ctx)
	if span == nil {
		return
	}
	span.Tags.Merge(tags)
}

func End(ctx sc.Context, tags ...string) (tracer Tracer, finished bool) {
	span := LoadSpan(ctx)
	if span == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: trace end failed cause not begin")))
		return
	}
	tagsLen := len(tags)
	if tagsLen > 0 && tagsLen%2 != 0 {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: trace end failed cause tags is invalid")))
		return
	}
	span.Tags.Merge(tags)

	span.End = time.Now()
	span.Latency = span.End.Sub(span.Beg)
	span.Handling = span.Latency - span.Waiting

	finished = span.parent == nil
	if finished {
		t := ctx.Value(contextTracerIdKey)
		if t == nil {
			panic(fmt.Sprintf("%+v", errors.Warning("fns: trace end failed cause there is no tracer in context")))
			return
		}
		tid, ok := t.([]byte)
		if !ok {
			panic(fmt.Sprintf("%+v", errors.Warning("fns: trace end failed cause type of tracer id in context is not []byte")))
			return
		}
		tracer = Tracer{
			Id:   tid,
			Span: span,
		}
	}
	return
}

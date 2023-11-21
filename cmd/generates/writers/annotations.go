package writers

import (
	"context"
	"github.com/aacfactory/gcg"
)

type FnAnnotationCodeWriter interface {
	Annotation() (annotation string)
	HandleBefore(ctx context.Context, params []string) (code gcg.Code, err error)
	HandleAfter(ctx context.Context, params []string) (code gcg.Code, err error)
	ProxyBefore(ctx context.Context, params []string) (code gcg.Code, err error)
	ProxyAfter(ctx context.Context, params []string) (code gcg.Code, err error)
}

type FnAnnotationCodeWriters []FnAnnotationCodeWriter

func (writers FnAnnotationCodeWriters) Get(annotation string) (w FnAnnotationCodeWriter, has bool) {
	for _, writer := range writers {
		if writer.Annotation() == annotation {
			w = writer
			has = true
			return
		}
	}
	return
}

const (
	fnAnnotationCodeWritersContextKey = "@fns:generates:writers:annotations"
)

func WithFnAnnotationCodeWriters(ctx context.Context, writers FnAnnotationCodeWriters) context.Context {
	return context.WithValue(ctx, fnAnnotationCodeWritersContextKey, writers)
}

func LoadFnAnnotationCodeWriters(ctx context.Context) (w FnAnnotationCodeWriters) {
	v := ctx.Value(fnAnnotationCodeWritersContextKey)
	if v == nil {
		w = make(FnAnnotationCodeWriters, 0, 1)
		return
	}
	ok := false
	w, ok = v.(FnAnnotationCodeWriters)
	if !ok {
		w = make(FnAnnotationCodeWriters, 0, 1)
	}
	return
}

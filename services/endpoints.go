package services

import (
	"context"
	"github.com/aacfactory/fns/services/documents"
)

type Handler interface {
	Handle(ctx Request) (v interface{}, err error)
}

type HandlerFunc func(ctx Request) (v interface{}, err error)

func (f HandlerFunc) Handle(ctx Request) (v interface{}, err error) {
	v, err = f(ctx)
	return
}

type Endpoint interface {
	Name() (name string)
	Internal() (ok bool)
	Document() (document *documents.Document)
	Handler
	Shutdown(ctx context.Context)
}

type Endpoints interface {
	Request(ctx context.Context, name []byte, fn []byte, argument Argument, options ...RequestOption) (response Response, err error)
}

type HostEndpoints interface {
	Endpoints
	Documents() (v *documents.Documents)
}

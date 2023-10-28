package services

import (
	"context"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/services/documents"
)

type Endpoint interface {
	Name() (name string)
	Internal() (ok bool)
	Document() (document *documents.Document)
	Handle(ctx context.Context, fn []byte, argument Argument) (v interface{}, err error)
	Close()
}

type Endpoints interface {
	Documents() (v documents.Documents)
	Request(ctx context.Context, name []byte, fn []byte, argument Argument, options ...RequestOption) (future futures.Future)
}

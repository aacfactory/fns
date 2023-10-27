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
}

type Endpoints interface {
	Request(ctx context.Context, service []byte, fn []byte, argument Argument, options ...RequestOption) (future futures.Future, err error)
}

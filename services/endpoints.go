package services

import (
	"context"
	"github.com/aacfactory/fns/services/documents"
)

type Endpoint interface {
	Name() (name string)
	Internal() (ok bool)
	Document() (document *documents.Document)
	Handle(ctx Request) (v interface{}, err error)
	Close()
}

type Endpoints interface {
	Documents() (v *documents.Documents)
	Request(ctx context.Context, name []byte, fn []byte, argument Argument, options ...RequestOption) (response Response, err error)
}

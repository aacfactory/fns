package development

import (
	"context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
)

type Endpoint struct {
	client transports.Client
}

func (endpoint *Endpoint) Name() (name string) {
	//TODO implement me
	panic("implement me")
}

func (endpoint *Endpoint) Internal() (ok bool) {
	//TODO implement me
	panic("implement me")
}

func (endpoint *Endpoint) Document() (document *documents.Document) {
	//TODO implement me
	panic("implement me")
}

func (endpoint *Endpoint) Handle(ctx services.Request) (v interface{}, err error) {
	//TODO implement me
	panic("implement me")
}

func (endpoint *Endpoint) Shutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

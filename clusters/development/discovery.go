package development

import (
	"context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
)

type Discovery struct {
	address []byte
	client  transports.Client
}

func (discovery *Discovery) Endpoints(ctx context.Context) (infos []services.EndpointInfo) {
	//TODO implement me
	panic("implement me")
}

func (discovery *Discovery) Get(ctx context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	//TODO implement me
	panic("implement me")
}

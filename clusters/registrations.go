package clusters

import (
	"context"
	"github.com/aacfactory/fns/services"
)

type Registrations struct {
}

func (rs *Registrations) Get(ctx context.Context, service []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {

	return
}

package development

import (
	"context"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
)

func NewDiscovery(address string, dialer transports.Dialer, signature signatures.Signature) services.Discovery {
	return &Discovery{
		address:   []byte(address),
		dialer:    dialer,
		signature: signature,
	}
}

type Discovery struct {
	address   []byte
	dialer    transports.Dialer
	signature signatures.Signature
}

func (discovery *Discovery) Endpoints(ctx context.Context) (infos []services.EndpointInfo) {
	//TODO implement me
	panic("implement me")
}

func (discovery *Discovery) Get(ctx context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	//TODO implement me
	panic("implement me")
}

// +-------------------------------------------------------------------------------------------------------------------+

var (
	discoveryHandlePathPrefix = []byte("/development/discovery/")
)

func NewDiscoveryHandler(signature signatures.Signature) transports.Handler {
	return &DiscoveryHandler{
		signature: signature,
	}
}

type DiscoveryHandler struct {
	signature signatures.Signature
}

func (handler *DiscoveryHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	//TODO implement me
	panic("implement me")
}

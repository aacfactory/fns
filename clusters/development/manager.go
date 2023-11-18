package development

import (
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
)

func NewManager(local services.EndpointsManager, log logs.Logger, address string, worker workers.Workers, dialer transports.Dialer, signature signatures.Signature) (v services.EndpointsManager) {

	return
}

type Manager struct {
}

func (manager *Manager) Info() (infos services.EndpointInfos) {
	//TODO implement me
	panic("implement me")
}

func (manager *Manager) PublicFnAddress(ctx context.Context, endpoint []byte, fnName []byte, options ...services.EndpointGetOption) (address string, has bool) {

	return
}

func (manager *Manager) Get(ctx context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	// local. not exist, remote
	//TODO implement me
	panic("implement me")
}

func (manager *Manager) Request(ctx context.Context, name []byte, fn []byte, param interface{}, options ...services.RequestOption) (response services.Response, err error) {
	//TODO implement me
	panic("implement me")
}

func (manager *Manager) Add(service services.Service) (err error) {
	//TODO implement me
	panic("implement me")
}

func (manager *Manager) Listen(ctx context.Context) (err error) {
	//TODO implement me
	panic("implement me")
}

func (manager *Manager) Shutdown(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

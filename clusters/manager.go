package clusters

import (
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"sync"
	"time"
)

func NewManager(cluster Cluster, local services.EndpointsManager, log logs.Logger, dialer transports.Dialer, signature signatures.Signature) services.EndpointsManager {
	v := &Manager{
		cluster:       cluster,
		local:         local,
		log:           log.With("cluster", "endpoints"),
		dialer:        dialer,
		signature:     signature,
		registrations: make(Registrations, 0, 1),
		locker:        sync.RWMutex{},
	}
	return v
}

type Manager struct {
	log           logs.Logger
	cluster       Cluster
	local         services.EndpointsManager
	dialer        transports.Dialer
	signature     signatures.Signature
	registrations Registrations
	locker        sync.RWMutex
}

func (manager *Manager) Add(service services.Service) (err error) {
	err = manager.local.Add(service)
	if err != nil {
		return
	}
	info, infoErr := NewService(service.Name(), service.Internal(), service.Document())
	if infoErr != nil {
		err = errors.Warning("fns: create cluster service info failed").WithCause(infoErr).WithMeta("service", service.Name())
		return
	}
	manager.cluster.AddService(info)
	return
}

func (manager *Manager) Info() (infos services.EndpointInfos) {
	manager.locker.RLock()
	defer manager.locker.RUnlock()
	infos = make([]services.EndpointInfo, 0, 1)
	for _, registration := range manager.registrations {
		for _, value := range registration.values {
			infos = append(infos, services.EndpointInfo{
				Id:       value.id,
				Name:     value.name,
				Version:  value.version,
				Internal: value.internal,
				Document: value.document,
			})
		}
	}
	return
}

func (manager *Manager) Registration(ctx context.Context, name []byte, options ...services.EndpointGetOption) (info Service, has bool, err error) {
	// todo return endpoint  Registration  with node info
	// used by proxies
	// 重新document
	// documents > endpoint
	// document > service
	// fn > fn
	return
}

func (manager *Manager) Get(ctx context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	//TODO implement me
	panic("implement me")
}

func (manager *Manager) Request(ctx context.Context, name []byte, fn []byte, param interface{}, options ...services.RequestOption) (response services.Response, err error) {
	//TODO implement me
	panic("implement me")
}

func (manager *Manager) Listen(ctx context.Context) (err error) {
	// cluster.join
	// watching
	// local.listen
	return
}

func (manager *Manager) Shutdown(ctx context.Context) {

	return
}

func (manager *Manager) watching() {
	go func(eps *Manager) {
		for {
			event, ok := <-eps.cluster.NodeEvents()
			if !ok {
				break
			}
			switch event.Kind {
			case Add:
				endpoints := make(Endpoints, 0, 1)
				client, clientErr := eps.dialer.Dial(bytex.FromString(event.Node.Address))
				if clientErr != nil {
					if eps.log.WarnEnabled() {
						eps.log.Warn().
							With("cluster", "registrations").
							Cause(errors.Warning(fmt.Sprintf("fns: dial %s failed", event.Node.Address)).WithMeta("address", event.Node.Address).WithCause(clientErr)).
							Message(fmt.Sprintf("fns: dial %s failed", event.Node.Address))
					}
					break
				}
				// check health
				active := false
				for i := 0; i < 10; i++ {
					ctx, cancel := sc.WithTimeout(context.TODO(), 2*time.Second)
					if runtime.CheckHealth(ctx, client) {
						active = true
						cancel()
						break
					}
					cancel()
					time.Sleep(1 * time.Second)
				}
				if !active {
					break
				}
				// get document
				for _, endpoint := range event.Node.Services {
					document, documentErr := endpoint.Document()
					if documentErr != nil {
						if eps.log.WarnEnabled() {
							eps.log.Warn().
								With("cluster", "registrations").
								Cause(errors.Warning("fns: get endpoint document failed").WithMeta("address", event.Node.Address).WithCause(documentErr)).
								Message(fmt.Sprintf("fns: dial %s failed", event.Node.Address))
						}
						continue
					}
					endpoints = append(endpoints, NewEndpoint(event.Node.Id, event.Node.Version, endpoint.Name, endpoint.Internal, document, client, eps.signature))
				}
				eps.locker.Lock()
				for _, endpoint := range endpoints {
					eps.registrations = eps.registrations.Add(endpoint)
				}
				eps.locker.Unlock()
				break
			case Remove:
				eps.locker.Lock()
				for _, endpoint := range event.Node.Services {
					eps.registrations = eps.registrations.Remove(endpoint.Name, event.Node.Id)
				}
				eps.locker.Unlock()
				break
			default:
				break
			}
		}
	}(manager)
	return
}

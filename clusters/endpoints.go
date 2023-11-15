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

func NewEndpoints(local services.Endpoints, log logs.Logger, dialer transports.Dialer, signature signatures.Signature, events <-chan NodeEvent) services.EndpointsManager {
	v := &endpoints{
		local:     local,
		log:       log.With("cluster", "endpoints"),
		dialer:    dialer,
		signature: signature,
		names:     make(NamedRegistrations, 0, 1),
		locker:    sync.RWMutex{},
		events:    events,
	}
	return v
}

type endpoints struct {
	log       logs.Logger
	local     services.Endpoints
	dialer    transports.Dialer
	signature signatures.Signature
	names     NamedRegistrations
	locker    sync.RWMutex
	events    <-chan NodeEvent
}

func (eps *endpoints) Add(service services.Service) (err error) {

	return
}

func (eps *endpoints) Info() (infos services.EndpointInfos) {
	eps.locker.RLock()
	defer eps.locker.RUnlock()
	infos = make([]services.EndpointInfo, 0, 1)
	for _, name := range eps.names {
		for _, value := range name.values {
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

func (eps *endpoints) Get(ctx context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	//TODO implement me
	panic("implement me")
}

func (eps *endpoints) Request(ctx context.Context, name []byte, fn []byte, param interface{}, options ...services.RequestOption) (response services.Response, err error) {
	//TODO implement me
	panic("implement me")
}

func (eps *endpoints) Listen(ctx context.Context) (err error) {
	// watching
	// local.listen
	return
}

func (eps *endpoints) Shutdown(ctx context.Context) {

	return
}

func (eps *endpoints) watching() {
	go func(eps *endpoints) {
		for {
			event, ok := <-eps.events
			if !ok {
				break
			}
			switch event.Kind {
			case Add:
				registrations := make([]*Registration, 0, 1)
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
				for _, endpoint := range event.Node.Endpoints {
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
					r := NewRegistration(event.Node.Id, endpoint.Name, event.Node.Version, endpoint.Internal, document, client, eps.signature)
					registrations = append(registrations, r)
				}
				eps.locker.Lock()
				for _, registration := range registrations {
					eps.names = eps.names.Add(registration)
				}
				eps.locker.Unlock()
				break
			case Remove:
				eps.locker.Lock()
				for _, endpoint := range event.Node.Endpoints {
					eps.names = eps.names.Remove(endpoint.Name, event.Node.Id)
				}
				eps.locker.Unlock()
				break
			default:
				break
			}
		}
	}(eps)
	return
}

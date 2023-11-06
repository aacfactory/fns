package clusters

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/handlers"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"sync"
	"time"
)

func NewDiscovery(log logs.Logger, dialer transports.Dialer, signature signatures.Signature, events <-chan NodeEvent) services.Discovery {
	v := &Discovery{
		log:       log.With("cluster", "discovery"),
		dialer:    dialer,
		signature: signature,
		names:     make(NamedRegistrations, 0, 1),
		locker:    new(sync.RWMutex),
		events:    events,
	}
	v.watching()
	return v
}

type Discovery struct {
	log       logs.Logger
	dialer    transports.Dialer
	signature signatures.Signature
	names     NamedRegistrations
	locker    *sync.RWMutex
	events    <-chan NodeEvent
}

func (discovery *Discovery) Endpoints(_ context.Context) (infos []services.EndpointInfo) {
	discovery.locker.RLock()
	defer discovery.locker.RUnlock()
	infos = make([]services.EndpointInfo, 0, 1)
	for _, name := range discovery.names {
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

func (discovery *Discovery) Get(_ context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	discovery.locker.RLock()
	named, exist := discovery.names.Get(name)
	if !exist {
		discovery.locker.RUnlock()
		return
	}
	opt := services.EndpointGetOptions{}
	for _, option := range options {
		option(&opt)
	}
	if id := opt.Id(); len(id) > 0 {
		endpoint, has = named.Get(id)
		discovery.locker.RUnlock()
		return
	}
	intervals := opt.Versions()
	if len(intervals) == 0 {
		endpoint, has = named.MaxOne()
	} else {
		interval, got := intervals.Get(name)
		if got {
			endpoint, has = named.Range(interval)
		} else {
			endpoint, has = named.MaxOne()
		}
	}
	discovery.locker.RUnlock()
	return
}

func (discovery *Discovery) watching() {
	go func(discovery *Discovery) {
		for {
			event, ok := <-discovery.events
			if !ok {
				break
			}
			switch event.Kind {
			case Add:
				registrations := make([]*Registration, 0, 1)
				client, clientErr := discovery.dialer.Dial(bytex.FromString(event.Node.Address))
				if clientErr != nil {
					if discovery.log.WarnEnabled() {
						discovery.log.Warn().
							With("cluster", "registrations").
							Cause(errors.Warning(fmt.Sprintf("fns: dial %s failed", event.Node.Address)).WithMeta("address", event.Node.Address).WithCause(clientErr)).
							Message(fmt.Sprintf("fns: dial %s failed", event.Node.Address))
					}
					break
				}
				// check health
				active := false
				for i := 0; i < 10; i++ {
					ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
					if handlers.CheckHealth(ctx, client) {
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
						if discovery.log.WarnEnabled() {
							discovery.log.Warn().
								With("cluster", "registrations").
								Cause(errors.Warning("fns: get endpoint document failed").WithMeta("address", event.Node.Address).WithCause(documentErr)).
								Message(fmt.Sprintf("fns: dial %s failed", event.Node.Address))
						}
						continue
					}
					r := NewRegistration(bytex.FromString(event.Node.Id), bytex.FromString(endpoint.Name), event.Node.Version, endpoint.Internal, document, client, discovery.signature)
					registrations = append(registrations, r)
				}
				discovery.locker.Lock()
				for _, registration := range registrations {
					discovery.names = discovery.names.Add(registration)
				}
				discovery.locker.Unlock()
				break
			case Remove:
				discovery.locker.Lock()
				for _, endpoint := range event.Node.Endpoints {
					discovery.names = discovery.names.Remove(bytex.FromString(endpoint.Name), bytex.FromString(event.Node.Id))
				}
				discovery.locker.Unlock()
				break
			default:
				break
			}
		}
	}(discovery)
	return
}

package clusters

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"sync"
)

type Registrations struct {
	log       logs.Logger
	cluster   Cluster
	dialer    transports.Dialer
	signature signatures.Signature
	closeFn   context.CancelFunc
	closedCh  chan struct{}
	node      *Node
	names     NamedRegistrations
	locker    sync.RWMutex
}

// Add
// 在services.Add后立即调用
func (rs *Registrations) Add(name string, internal bool, listenable bool) {
	rs.node.Endpoints = append(rs.node.Endpoints, EndpointInfo{
		Name:       name,
		Internal:   internal,
		Listenable: listenable,
	})
	return
}

func (rs *Registrations) Find(_ context.Context, name []byte, options ...services.EndpointGetOption) (registration *Registration, has bool) {
	rs.locker.RLock()
	named, exist := rs.names.Get(name)
	if !exist {
		rs.locker.RUnlock()
		return
	}
	opt := services.EndpointGetOptions{}
	for _, option := range options {
		option(&opt)
	}
	if id := opt.Id(); len(id) > 0 {
		registration, has = named.Get(id)
		rs.locker.RUnlock()
		return
	}
	intervals := opt.Versions()
	if len(intervals) == 0 {
		registration, has = named.MaxOne()
	} else {
		interval, got := intervals.Get(name)
		if got {
			registration, has = named.Range(interval)
		} else {
			registration, has = named.MaxOne()
		}
	}
	rs.locker.RUnlock()
	return
}

func (rs *Registrations) Get(ctx context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	endpoint, has = rs.Find(ctx, name, options...)
	return
}

func (rs *Registrations) Watching(ctx context.Context) {
	ctx, rs.closeFn = context.WithCancel(ctx)
	go func(ctx context.Context, rs *Registrations) {
		closed := false
		for {
			select {
			case <-ctx.Done():
				closed = true
				break
			case event, ok := <-rs.cluster.NodeEvents():
				if !ok {
					closed = true
					break
				}

				switch event.Kind {
				case Add:
					registrations := make([]*Registration, 0, 1)
					client, clientErr := rs.dialer.Dial(event.Node.Address)
					if clientErr != nil {
						if rs.log.WarnEnabled() {
							rs.log.Warn().
								With("cluster", "registrations").
								Cause(errors.Warning(fmt.Sprintf("fns: dialer %s failed", event.Node.Address)).WithMeta("address", event.Node.Address).WithCause(clientErr)).
								Message(fmt.Sprintf("fns: dialer %s failed", event.Node.Address))
						}
						break
					}
					// get document
					var document documents.Documents
					for _, endpoint := range event.Node.Endpoints {
						r := NewRegistration(bytex.FromString(event.Node.Id), bytex.FromString(endpoint.Name), event.Node.Version, client, rs.signature)
						registrations = append(registrations, r)
					}
					rs.locker.Lock()
					for _, registration := range registrations {
						rs.names = rs.names.Add(registration)
					}
					rs.locker.Unlock()
					break
				case Remove:
					rs.locker.Lock()
					for _, endpoint := range event.Node.Endpoints {
						rs.names = rs.names.Remove(bytex.FromString(endpoint.Name), bytex.FromString(event.Node.Id))
					}
					rs.locker.Unlock()
					break
				}

				break
			}
			if closed {
				break
			}
		}
	}(ctx, rs)
	return
}

func (node Node) makeRegistration(ctx context.Context) (v *Registration, err error) {

	return
}

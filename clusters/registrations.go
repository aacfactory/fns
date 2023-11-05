package clusters

import (
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/handlers"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"strings"
	"sync"
	"time"
)

type Options struct {
	Id      string
	Name    string
	Version versions.Version
	Port    int
	Log     logs.Logger
	Dialer  transports.Dialer
	Config  Config
}

func New(options Options) (rs *Registrations, shared shareds.Shared, barrier barriers.Barrier, err error) {
	// host
	hostRetrieverName := strings.TrimSpace(options.Config.HostRetriever)
	if hostRetrieverName == "" {
		hostRetrieverName = "default"
	}
	hostRetriever, hasHostRetriever := getHostRetriever(hostRetrieverName)
	if !hasHostRetriever {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(fmt.Errorf("host retriever was not found")).WithMeta("name", hostRetrieverName)
		return
	}
	host, hostErr := hostRetriever()
	if hostErr != nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(hostErr)
		return
	}
	// node
	node := &Node{
		Id:        options.Id,
		Name:      options.Name,
		Version:   options.Version,
		Address:   fmt.Sprintf("%s:%d", host, options.Port),
		Endpoints: make([]EndpointInfo, 0, 1),
	}
	// signature
	secret := options.Config.Secret
	if secret == "" {
		secret = "FNS+-"
	}
	signature := signatures.HMAC([]byte(secret))
	// cluster
	cluster, hasCluster := loadCluster(options.Config.Name)
	if !hasCluster {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(fmt.Errorf("cluster was not found")).WithMeta("name", options.Config.Name)
		return
	}
	if options.Config.Option == nil && len(options.Config.Option) < 2 {
		options.Config.Option = []byte{'{', '}'}
	}
	clusterConfig, clusterConfigErr := configures.NewJsonConfig(options.Config.Option)
	if clusterConfigErr != nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(clusterConfigErr).WithMeta("name", options.Config.Name)
		return
	}
	clusterErr := cluster.Construct(ClusterOptions{
		Log:    options.Log.With("cluster", options.Config.Name),
		Config: clusterConfig,
	})
	if clusterErr != nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(clusterErr).WithMeta("name", options.Config.Name)
		return
	}
	// shared
	shared = cluster.Shared()
	if shared == nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(fmt.Errorf("cluster return a nil shared")).WithMeta("name", options.Config.Name)
		return
	}
	sharedConfigBytes := options.Config.Shared
	if len(sharedConfigBytes) == 0 {
		sharedConfigBytes = []byte{'{', '}'}
	}
	sharedConfig, sharedConfigErr := configures.NewJsonConfig(sharedConfigBytes)
	if sharedConfigErr != nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(sharedConfigErr).WithMeta("name", options.Config.Name)
		return
	}
	sharedErr := shared.Construct(shareds.Options{
		Log:    options.Log.With("shared", "cluster"),
		Config: sharedConfig,
	})
	if sharedErr != nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(sharedErr).WithMeta("name", options.Config.Name)
		return
	}
	// barrier
	barrier = cluster.Barrier()
	if barrier == nil {
		barrier = NewDefaultBarrier()
	}
	barrierConfigBytes := options.Config.Barrier
	if len(barrierConfigBytes) == 0 {
		barrierConfigBytes = []byte{'{', '}'}
	}
	barrierConfig, barrierConfigErr := configures.NewJsonConfig(barrierConfigBytes)
	if barrierConfigErr != nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(barrierConfigErr).WithMeta("name", options.Config.Name)
		return
	}
	barrierErr := barrier.Construct(barriers.Options{
		Log:    options.Log.With("barrier", "cluster"),
		Config: barrierConfig,
	})
	if barrierErr != nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(barrierErr).WithMeta("name", options.Config.Name)
		return
	}
	// rs
	rs = &Registrations{
		log:       options.Log.With("fns", "registrations"),
		cluster:   cluster,
		dialer:    options.Dialer,
		signature: signature,
		closeFn:   nil,
		closedCh:  make(chan struct{}, 1),
		node:      node,
		names:     make(NamedRegistrations, 0, 1),
		locker:    sync.RWMutex{},
	}
	return
}

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

func (rs *Registrations) Add(name string, internal bool, document *documents.Document) (err error) {
	info, infoErr := NewEndpointInfo(name, internal, document)
	if infoErr != nil {
		err = errors.Warning("fns: registrations add endpoint info failed").WithCause(infoErr)
		return
	}
	rs.node.Endpoints = append(rs.node.Endpoints, info)
	return
}

func (rs *Registrations) Shared() shareds.Shared {
	return rs.cluster.Shared()
}

func (rs *Registrations) Signature() signatures.Signature {
	return rs.signature
}

func (rs *Registrations) Endpoints(_ context.Context) (infos []services.EndpointInfo) {
	rs.locker.RLock()
	defer rs.locker.RUnlock()
	infos = make([]services.EndpointInfo, 0, 1)
	for _, name := range rs.names {
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

func (rs *Registrations) Get(_ context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
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
		endpoint, has = named.Get(id)
		rs.locker.RUnlock()
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
	rs.locker.RUnlock()
	return
}

func (rs *Registrations) Watching(ctx context.Context) (err error) {
	joinErr := rs.cluster.Join(ctx, *rs.node)
	if joinErr != nil {
		err = errors.Warning("fns: watching registrations failed").WithCause(joinErr)
		return
	}
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
					client, clientErr := rs.dialer.Dial(bytex.FromString(event.Node.Address))
					if clientErr != nil {
						if rs.log.WarnEnabled() {
							rs.log.Warn().
								With("cluster", "registrations").
								Cause(errors.Warning(fmt.Sprintf("fns: dial %s failed", event.Node.Address)).WithMeta("address", event.Node.Address).WithCause(clientErr)).
								Message(fmt.Sprintf("fns: dial %s failed", event.Node.Address))
						}
						break
					}
					// check health
					active := false
					for i := 0; i < 10; i++ {
						if handlers.CheckHealth(ctx, client) {
							active = true
							break
						}
						time.Sleep(1 * time.Second)
					}
					if !active {
						break
					}
					// get document
					for _, endpoint := range event.Node.Endpoints {
						document, documentErr := endpoint.Document()
						if documentErr != nil {
							if rs.log.WarnEnabled() {
								rs.log.Warn().
									With("cluster", "registrations").
									Cause(errors.Warning("fns: get endpoint document failed").WithMeta("address", event.Node.Address).WithCause(documentErr)).
									Message(fmt.Sprintf("fns: dial %s failed", event.Node.Address))
							}
							continue
						}
						r := NewRegistration(bytex.FromString(event.Node.Id), bytex.FromString(endpoint.Name), event.Node.Version, endpoint.Internal, document, client, rs.signature)
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
		_ = rs.cluster.Leave(ctx)
		rs.closedCh <- struct{}{}
		close(rs.closedCh)
	}(ctx, rs)
	return
}

func (rs *Registrations) Shutdown(ctx context.Context) {
	rs.closeFn()
	select {
	case <-ctx.Done():
		break
	case <-rs.closedCh:
		break
	}
	return
}

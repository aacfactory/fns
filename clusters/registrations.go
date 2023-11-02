package clusters

import (
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/handlers"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
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

func New(options Options) (rs *Registrations, err error) {
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
	cluster, hasCluster := loadCluster(options.Config.Kind)
	if !hasCluster {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(fmt.Errorf("cluster was not found")).WithMeta("name", options.Config.Kind)
		return
	}
	if options.Config.Option == nil && len(options.Config.Option) < 2 {
		options.Config.Option = []byte{'{', '}'}
	}
	clusterConfig, clusterConfigErr := configures.NewJsonConfig(options.Config.Option)
	if clusterConfigErr != nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(clusterConfigErr).WithMeta("name", options.Config.Kind)
		return
	}
	clusterErr := cluster.Construct(ClusterOptions{
		Log:    options.Log.With("cluster", options.Config.Kind),
		Config: clusterConfig,
	})
	if clusterErr != nil {
		err = errors.Warning("fns: new cluster registrations failed").WithCause(clusterErr).WithMeta("name", options.Config.Kind)
		return
	}

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

// Registrations
// implement discovery + handler
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
					client, clientErr := rs.dialer.Dial(event.Node.Address)
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
					document, documentErr := handlers.FetchDocuments(ctx, client)
					if documentErr != nil {
						if rs.log.WarnEnabled() {
							rs.log.Warn().
								With("cluster", "registrations").
								Cause(errors.Warning(fmt.Sprintf("fns: get documents from %s failed", event.Node.Address)).WithMeta("address", event.Node.Address).WithCause(clientErr)).
								Message(fmt.Sprintf("fns: get documents from %s failed", event.Node.Address))
						}
						break
					}
					for _, endpoint := range event.Node.Endpoints {
						var doc *documents.Document
						if document != nil {
							doc = document.Get(bytex.FromString(endpoint.Name))
						}
						r := NewRegistration(bytex.FromString(event.Node.Id), bytex.FromString(endpoint.Name), event.Node.Version, doc, client, rs.signature)
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

func (rs *Registrations) Close() {
	rs.closeFn()
	<-rs.closedCh
	return
}

package clusters

import (
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/clusters/development"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"strings"
)

type ClusterOptions struct {
	Log     logs.Logger
	Config  configures.Config
	Id      string
	Name    string
	Version versions.Version
	Address string
}

type Cluster interface {
	Construct(options ClusterOptions) (err error)
	AddService(service Service)
	Join(ctx context.Context) (err error)
	Leave(ctx context.Context) (err error)
	NodeEvents() (events <-chan NodeEvent)
	Shared() (shared shareds.Shared)
}

type ClusterBuilderOptions struct {
	Config configures.Config
	Log    logs.Logger
}

var (
	clusterMap = make(map[string]Cluster)
)

func RegisterCluster(name string, cluster Cluster) {
	clusterMap[name] = cluster
}

func loadCluster(name string) (cluster Cluster, has bool) {
	cluster, has = clusterMap[name]
	return
}

type Options struct {
	Id      string
	Name    string
	Version versions.Version
	Port    int
	Log     logs.Logger
	Local   services.EndpointsManager
	Dialer  transports.Dialer
	Config  Config
}

func New(options Options) (manager services.EndpointsManager, barrier barriers.Barrier, handlers []transports.MuxHandler, err error) {
	// dev
	if options.Config.Name == developmentName {
		if options.Config.DevMode {
			err = errors.Warning("fns: new cluster failed").WithCause(fmt.Errorf("dev cluster can not use dev mode"))
			return
		}
		manager, barrier, handlers, err = NewDevelopment(options)
		return
	}
	// host
	hostRetrieverName := strings.TrimSpace(options.Config.HostRetriever)
	if hostRetrieverName == "" {
		hostRetrieverName = "default"
	}
	hostRetriever, hasHostRetriever := getHostRetriever(hostRetrieverName)
	if !hasHostRetriever {
		err = errors.Warning("fns: new cluster failed").WithCause(fmt.Errorf("host retriever was not found")).WithMeta("name", hostRetrieverName)
		return
	}
	host, hostErr := hostRetriever()
	if hostErr != nil {
		err = errors.Warning("fns: new cluster failed").WithCause(hostErr)
		return
	}
	// signature
	signature := NewSignature(options.Config.Secret)
	// cluster
	cluster, hasCluster := loadCluster(options.Config.Name)
	if !hasCluster {
		err = errors.Warning("fns: new cluster failed").WithCause(fmt.Errorf("cluster was not found")).WithMeta("name", options.Config.Name)
		return
	}
	if options.Config.Option == nil && len(options.Config.Option) < 2 {
		options.Config.Option = []byte{'{', '}'}
	}
	clusterConfig, clusterConfigErr := configures.NewJsonConfig(options.Config.Option)
	if clusterConfigErr != nil {
		err = errors.Warning("fns: new cluster failed").WithCause(clusterConfigErr).WithMeta("name", options.Config.Name)
		return
	}
	clusterErr := cluster.Construct(ClusterOptions{
		Log:     options.Log.With("cluster", options.Config.Name),
		Config:  clusterConfig,
		Id:      options.Id,
		Name:    options.Name,
		Version: options.Version,
		Address: fmt.Sprintf("%s:%d", host, options.Port),
	})
	if clusterErr != nil {
		err = errors.Warning("fns: new cluster failed").WithCause(clusterErr).WithMeta("name", options.Config.Name)
		return
	}
	// barrier
	barrier = NewBarrier(options.Config.Barrier)
	// manager
	manager = NewManager(cluster, options.Local, options.Log, options.Dialer, signature)
	// handlers
	handlers = make([]transports.MuxHandler, 0, 1)
	handlers = append(handlers, NewInternalHandler(options.Local, signature))
	if options.Config.DevMode {
		// append dev handler
		handlers = append(handlers, development.NewHandler(signature, manager, cluster.Shared()))
	}
	return
}

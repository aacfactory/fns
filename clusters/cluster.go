package clusters

import (
	"context"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/logs"
)

type ClusterOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Cluster interface {
	Construct(options ClusterOptions) (err error)
	Join(ctx context.Context, node Node) (err error)
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

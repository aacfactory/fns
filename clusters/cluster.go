package clusters

import (
	"context"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/logs"
)

type Cluster interface {
	Join(ctx context.Context, node Node) (err error)
	Leave(ctx context.Context) (err error)
	Nodes(ctx context.Context) (nodes Nodes, err error)
	Shared() (shared shareds.Shared)
}

type ClusterBuilderOptions struct {
	Config configures.Config
	Log    logs.Logger
}

type ClusterBuilder func(options ClusterBuilderOptions) (cluster Cluster, err error)

var (
	builders = make(map[string]ClusterBuilder)
)

func RegisterClusterBuilder(name string, builder ClusterBuilder) {
	builders[name] = builder
}

func LoadClusterBuilder(name string) (builder ClusterBuilder, has bool) {
	builder, has = builders[name]
	return
}

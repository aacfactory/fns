package development

import (
	"context"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/shareds"
)

const (
	clusterBuilderName = "dev"
)

type Cluster struct {
	events chan clusters.NodeEvent
}

func (cluster *Cluster) Join(ctx context.Context, node clusters.Node) (err error) {
	return
}

func (cluster *Cluster) Leave(ctx context.Context) (err error) {
	return
}

func (cluster *Cluster) NodeEvents() (events <-chan clusters.NodeEvent) {
	events = cluster.events
	return
}

func (cluster *Cluster) Shared() (shared shareds.Shared) {
	return
}

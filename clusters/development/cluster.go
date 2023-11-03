package development

import (
	"context"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/shareds"
)

const (
	clusterName = "dev"
)

type Cluster struct {
	events chan clusters.NodeEvent
}

func (cluster *Cluster) Construct(options clusters.ClusterOptions) (err error) {
	return
}

func (cluster *Cluster) Join(_ context.Context, _ clusters.Node) (err error) {
	return
}

func (cluster *Cluster) Leave(_ context.Context) (err error) {
	return
}

func (cluster *Cluster) NodeEvents() (events <-chan clusters.NodeEvent) {
	events = cluster.events
	return
}

func (cluster *Cluster) Shared() (shared shareds.Shared) {
	return
}

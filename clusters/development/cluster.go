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
}

func (cluster *Cluster) Join(ctx context.Context) (err error) {
	return
}

func (cluster *Cluster) Leave(ctx context.Context) (err error) {
	return
}

func (cluster *Cluster) Nodes(ctx context.Context) (nodes clusters.Nodes, err error) {
	return
}

func (cluster *Cluster) Shared() (shared shareds.Shared) {
	return
}

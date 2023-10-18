package clusters

import (
	"context"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/logs"
)

type Node struct {
	Id      string           `json:"id"`
	Name    string           `json:"name"`
	Version versions.Version `json:"version"`
	Address string           `json:"address"`
}

type Nodes []*Node

func (nodes Nodes) Len() int {
	return len(nodes)
}

func (nodes Nodes) Less(i, j int) bool {
	return nodes[i].Id < nodes[j].Id
}

func (nodes Nodes) Swap(i, j int) {
	nodes[i], nodes[j] = nodes[j], nodes[i]
	return
}

type Cluster interface {
	Join(ctx context.Context) (err error)
	Leave(ctx context.Context) (err error)
	Nodes(ctx context.Context) (nodes Nodes, err error)
	Shared() (shared shareds.Shared)
}

type ClusterBuilderOptions struct {
	Config     configures.Config
	Log        logs.Logger
	AppId      string
	AppName    string
	AppVersion versions.Version
}

type ClusterBuilder func(options ClusterBuilderOptions) (cluster Cluster, err error)

var (
	builders = make(map[string]ClusterBuilder)
)

func RegisterClusterBuilder(name string, builder ClusterBuilder) {
	builders[name] = builder
}

func GetClusterBuilder(name string) (builder ClusterBuilder, has bool) {
	builder, has = builders[name]
	return
}

package clusters

import (
	"github.com/aacfactory/fns/commons/versions"
)

type EndpointInfo struct {
	Name     string `json:"name"`
	Internal bool   `json:"internal"`
}

type Node struct {
	Id        string           `json:"id"`
	Name      string           `json:"name"`
	Version   versions.Version `json:"version"`
	Address   string           `json:"address"`
	Endpoints []EndpointInfo   `json:"endpoints"`
}

const (
	Add    = NodeEventKind(1)
	Remove = NodeEventKind(2)
)

type NodeEventKind int

type NodeEvent struct {
	Kind NodeEventKind
	Node Node
}

type Nodes []Node

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

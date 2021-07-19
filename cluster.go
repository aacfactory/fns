package fns

import (
	"encoding/json"
	"github.com/aacfactory/cluster"
)

// +-------------------------------------------------------------------------------------------------------------------+

type ClusterConfig struct {
	Enable bool            `json:"enable,omitempty"`
	Kind   string          `json:"kind,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

var clusterRetriever ClusterRetriever = nil

type ClusterRetriever func(name string, tags []string, config []byte) (c cluster.Cluster, err error)

//RegisterClusterRetriever 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterClusterRetriever(fn ClusterRetriever) {
	clusterRetriever = fn
}

// +-------------------------------------------------------------------------------------------------------------------+

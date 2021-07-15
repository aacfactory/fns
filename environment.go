package fns

import "github.com/aacfactory/cluster"

type Environment interface {
	Config() (config Config)
	Discovery() (discovery cluster.ServiceDiscovery)
}

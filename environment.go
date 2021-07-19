package fns

import (
	"fmt"
	"github.com/aacfactory/cluster"
)

type Environment interface {
	ClusterMode() (ok bool)
	Config() (config Config)
	Discovery() (discovery cluster.ServiceDiscovery)
}

func newFnsEnvironment(config Config, discovery cluster.ServiceDiscovery) Environment {
	return &fnsEnvironment{
		config:    config,
		discovery: discovery,
	}
}

type fnsEnvironment struct {
	config    Config
	discovery cluster.ServiceDiscovery
}

func (env *fnsEnvironment) ClusterMode() (ok bool) {
	ok = env.discovery != nil
	return
}

func (env *fnsEnvironment) Config() (config Config) {
	config = env.config
	return
}

func (env *fnsEnvironment) Discovery() (discovery cluster.ServiceDiscovery) {
	if env.discovery == nil {
		panic(fmt.Errorf("fns is not in cluster mode"))
	}
	discovery = env.discovery
	return
}

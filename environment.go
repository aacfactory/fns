package fns

import "fmt"

type Environment interface {
	ClusterMode() (ok bool)
	Config() (config Config)
	Discovery() (discovery ServiceDiscovery)
}

func newFnsEnvironment(config Config, discovery ServiceDiscovery) Environment {
	return &fnsEnvironment{
		config:    config,
		discovery: discovery,
	}
}

type fnsEnvironment struct {
	config    Config
	discovery ServiceDiscovery
}

func (env *fnsEnvironment) ClusterMode() (ok bool) {
	ok = env.discovery != nil
	return
}

func (env *fnsEnvironment) Config() (config Config) {
	config = env.config
	return
}

func (env *fnsEnvironment) Discovery() (discovery ServiceDiscovery) {
	if env.discovery == nil {
		panic(fmt.Errorf("fns is not in cluster mode"))
	}
	discovery = env.discovery
	return
}

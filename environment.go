package fns

type Environment interface {
	Config() (config Config)
	Discovery() (discovery ServiceDiscovery)
}

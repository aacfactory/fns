package configs

import (
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/hooks"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/proxies"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
)

type WorkersConfig struct {
	Max            int `json:"max" yaml:"max,omitempty"`
	MaxIdleSeconds int `json:"maxIdleSeconds" yaml:"maxIdleSeconds,omitempty"`
}

type ProcsConfig struct {
	Min int `json:"min" yaml:"min,omitempty"`
}

type RuntimeConfig struct {
	Procs   ProcsConfig               `json:"procs,omitempty" yaml:"procs,omitempty"`
	Workers WorkersConfig             `json:"workers,omitempty" yaml:"workers,omitempty"`
	Shared  shareds.LocalSharedConfig `json:"shared,omitempty" yaml:"shared,omitempty"`
}

type Config struct {
	Runtime   RuntimeConfig     `json:"runtime,omitempty" yaml:"runtime,omitempty"`
	Log       logs.Config       `json:"log,omitempty" yaml:"log,omitempty"`
	Cluster   *clusters.Config  `json:"cluster,omitempty" yaml:"cluster,omitempty"`
	Transport transports.Config `json:"transport,omitempty" yaml:"transport,omitempty"`
	Proxy     proxies.Config    `json:"proxy,omitempty" yaml:"proxy,omitempty"`
	Services  services.Config   `json:"services,omitempty" yaml:"services,omitempty"`
	Hooks     hooks.Config      `json:"hooks,omitempty" yaml:"hooks,omitempty"`
}

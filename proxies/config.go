package proxies

import "github.com/aacfactory/fns/transports"

type DevConfig struct {
	Enable bool `json:"enable" yaml:"enable,omitempty"`
}

type Config struct {
	transports.Config
	Dev DevConfig `json:"dev" yaml:"dev,omitempty"`
}

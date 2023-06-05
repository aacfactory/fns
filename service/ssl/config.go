package ssl

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"net"
)

type Config interface {
	Build(options configures.Config) (err error)
	TLS() (serverTLS *tls.Config, clientTLS *tls.Config, err error)
	NewListener(inner net.Listener) (ln net.Listener)
}

var (
	configs = map[string]Config{
		"SSC":     &SSCConfig{},
		"DEFAULT": &DefaultConfig{},
		"GM":      &GMConfig{},
	}
)

func RegisterConfig(kind string, config Config) {
	if kind == "" || config == nil {
		return
	}
	_, has := configs[kind]
	if has {
		panic(fmt.Errorf("fns: regisger tls config failed for existed"))
	}
	configs[kind] = config
}

func GetConfig(kind string) (config Config, has bool) {
	config, has = configs[kind]
	return
}

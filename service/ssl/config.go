package ssl

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"net"
)

type ListenerFunc func(inner net.Listener) (ln net.Listener)

type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type Config interface {
	Build(options configures.Config) (err error)
	Server() (srvTLS *tls.Config, ln ListenerFunc)
	Client() (cliTLS *tls.Config, dialer Dialer)
}

var (
	configs = map[string]Config{
		"DEFAULT": &DefaultConfig{},
		"SSC":     &SSCConfig{},
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

package ssl

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"net"
)

type Listener func(inner net.Listener) (ln net.Listener)

type Dialer func(dialer *net.Dialer) (dialFunc func(ctx context.Context, network string, addr string) (conn net.Conn, err error))

type Config interface {
	Build(options configures.Config) (err error)
	Server() (srvTLS *tls.Config, ln Listener)
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

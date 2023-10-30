package proxies

import (
	"github.com/aacfactory/fns/transports"
)

type DefaultOptions struct {
	tr          transports.Transport
	middlewares []transports.Middlewares
	handlers    []transports.Handler
}

type Option func(*DefaultOptions) error

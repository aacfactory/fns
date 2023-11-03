package proxies

import (
	"github.com/aacfactory/fns/transports"
)

type Options struct {
	transport   transports.Transport
	middlewares []transports.Middleware
	handlers    []transports.Handler
}

type Option func(*Options) error

func Transport(transport transports.Transport) Option {
	return func(options *Options) error {
		options.transport = transport
		return nil
	}
}

func Middleware(middleware transports.Middleware) Option {
	return func(options *Options) error {
		options.middlewares = append(options.middlewares, middleware)
		return nil
	}
}

func Handler(handler transports.MuxHandler) Option {
	return func(options *Options) error {
		options.handlers = append(options.handlers, handler)
		return nil
	}
}

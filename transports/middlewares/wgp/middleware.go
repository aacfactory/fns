package wgp

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
)

func New() transports.Middleware {
	return &Middleware{}
}

// Middleware
// @get
type Middleware struct {
	log    logs.Logger
	enable bool
}

func (middle *Middleware) Name() string {
	return "wgp"
}

func (middle *Middleware) Construct(options transports.MiddlewareOptions) error {
	middle.log = options.Log
	config := Config{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		return errors.Warning("fns: construct wgp middleware failed").WithCause(configErr)
	}
	middle.enable = config.Enable
	return nil
}

func (middle *Middleware) Handler(next transports.Handler) transports.Handler {
	if middle.enable {
		return transports.HandlerFunc(func(writer transports.ResponseWriter, request transports.Request) {
			err := paths.WrapRequest(request)
			if err != nil {
				writer.Failed(errors.Warning("fns: wgp wrap request failed").WithCause(err))
				return
			}
			next.Handle(writer, request)
		})
	}
	return next
}

func (middle *Middleware) Close() {
	return
}

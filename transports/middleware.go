package transports

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
	"strings"
)

type MiddlewareOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Middleware interface {
	Name() string
	Construct(options MiddlewareOptions) error
	Handler(next Handler) Handler
	Close()
}

func WaveMiddlewares(log logs.Logger, config Config, middlewares []Middleware) (v Middlewares, err error) {
	for _, middleware := range middlewares {
		name := strings.TrimSpace(middleware.Name())
		mc, mcErr := config.Middleware(name)
		if mcErr != nil {
			err = errors.Warning("wave middlewares failed").WithCause(mcErr).WithMeta("middleware", name)
			return
		}
		constructErr := middleware.Construct(MiddlewareOptions{
			Log:    log.With("middleware", name),
			Config: mc,
		})
		if constructErr != nil {
			err = errors.Warning("wave middlewares failed").WithCause(constructErr).WithMeta("middleware", name)
			return
		}
	}
	v = middlewares
	return
}

type Middlewares []Middleware

func (middlewares Middlewares) Handler(handler Handler) Handler {
	if len(middlewares) == 0 {
		return handler
	}
	for i := len(middlewares) - 1; i > -1; i-- {
		handler = middlewares[i].Handler(handler)
	}
	return handler
}

func (middlewares Middlewares) Close() {
	for _, middleware := range middlewares {
		middleware.Close()
	}
}

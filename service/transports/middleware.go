package transports

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
	"strings"
)

type MiddlewareBuilderOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type MiddlewareBuilder interface {
	Name() string
	Build(options MiddlewareBuilderOptions) (middleware Middleware, err error)
}

type Middleware interface {
	Handler(next Handler) Handler
}

func WaveMiddlewares(log logs.Logger, config *Config, builders []MiddlewareBuilder) (v Middleware, err error) {
	middlewares := make([]Middleware, 0, 1)
	if len(builders) == 0 {
		v = &Middlewares{
			middlewares: middlewares,
		}
	}
	for _, builder := range builders {
		name := strings.TrimSpace(builder.Name())
		mc, mcErr := config.Middleware(name)
		if mcErr != nil {
			err = errors.Warning("wave middlewares failed").WithCause(mcErr).WithMeta("middleware", name)
			return
		}
		middleware, middlewaresErr := builder.Build(MiddlewareBuilderOptions{
			Log:    log.With("middleware", name),
			Config: mc,
		})
		if middlewaresErr != nil {
			err = errors.Warning("wave middlewares failed").WithCause(middlewaresErr).WithMeta("middleware", name)
			return
		}
		middlewares = append(middlewares, middleware)
	}
	v = &Middlewares{
		middlewares: middlewares,
	}
	return
}

type Middlewares struct {
	middlewares []Middleware
}

func (middlewares *Middlewares) Handler(handler Handler) Handler {
	if len(middlewares.middlewares) == 0 {
		return handler
	}
	for i := len(middlewares.middlewares) - 1; i > -1; i-- {
		handler = middlewares.middlewares[i].Handler(handler)
	}
	return handler
}

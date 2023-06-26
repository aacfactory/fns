package transports

import (
	"context"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/logs"
)

type MiddlewareOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Middleware interface {
	Name() (name string)
	Build(ctx context.Context, options MiddlewareOptions) (err error)
	Handler(next Handler) Handler
	Close() (err error)
}

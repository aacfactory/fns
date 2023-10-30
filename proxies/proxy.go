package proxies

import (
	"context"
	"github.com/aacfactory/logs"
)

type Options struct {
	Log    logs.Logger
	Config Config
}

type Proxy interface {
	Construct(options Options) (err error)
	Run(ctx context.Context) (err error)
	Shutdown(ctx context.Context)
}

package proxies

import (
	"context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/logs"
)

type ProxyOptions struct {
	Log     logs.Logger
	Config  Config
	Runtime *runtime.Runtime
}

type Proxy interface {
	Construct(options ProxyOptions) (err error)
	Run(ctx context.Context) (err error)
	Shutdown(ctx context.Context)
}

func New(options ...Option) (proxy Proxy, err error) {
	// todo handle cors
	return
}

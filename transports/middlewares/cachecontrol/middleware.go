package cachecontrol

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
)

func NewWithCache(cache Cache) transports.Middleware {
	return &Middleware{
		cache: cache,
	}
}

func New() transports.Middleware {
	return NewWithCache(new(DefaultCache))
}

// Middleware
// enable wgp(middleware) before this.
type Middleware struct {
	log    logs.Logger
	cache  Cache
	enable bool
	maxAge int
}

func (middleware *Middleware) Name() string {
	return "cachecontrol"
}

func (middleware *Middleware) Construct(options transports.MiddlewareOptions) (err error) {
	middleware.log = options.Log
	config := Config{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: construct cache control middleware failed").WithCause(configErr)
		return
	}
	middleware.enable = config.Enable
	middleware.maxAge = config.MaxAge
	if middleware.maxAge < 1 {
		middleware.maxAge = 60
	}
	return
}

func (middleware *Middleware) Handler(next transports.Handler) transports.Handler {
	if middleware.enable {
		return transports.HandlerFunc(func(writer transports.ResponseWriter, request transports.Request) {
			//cc := request.Header().Get(bytex.FromString(transports.CacheControlHeaderName))

		})
	}
	return next
}

func (middleware *Middleware) Close() {
	middleware.cache.Close()
	return
}

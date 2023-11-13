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
// todo
// 1. 内置（或者直接改handler），然后通过service的doc来做路由（或者endpointInfos，然后info里增加fn，来自doc），abstract service 不再提供document的
// 2. 不改endpointInfo，handler里放endpointInfo，然后path的第一个segment是命中就完事，abstract service也不改
// 关于internal的doc，参数和结果用any
// 修改argument，重名parameter。value支持Decode（v）的接口，tr的param实现Decode，然后不再用body
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

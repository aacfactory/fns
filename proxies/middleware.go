package proxies

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/transports"
	"sync"
)

func NewProxyMiddleware(rt *runtime.Runtime) transports.Middleware {
	return &ProxyMiddleware{
		rt:      rt,
		counter: new(sync.WaitGroup),
	}
}

type ProxyMiddleware struct {
	rt      *runtime.Runtime
	counter *sync.WaitGroup
}

func (middleware *ProxyMiddleware) Name() string {
	return "proxy"
}

func (middleware *ProxyMiddleware) Construct(_ transports.MiddlewareOptions) error {
	return nil
}

func (middleware *ProxyMiddleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
		running, upped := middleware.rt.Running()
		if !running {
			w.Header().Set(bytex.FromString(transports.ConnectionHeaderName), bytex.FromString(transports.CloseHeaderValue))
			w.Failed(ErrUnavailable)
			return
		}
		if !upped {
			w.Header().Set(bytex.FromString(transports.ResponseRetryAfterHeaderName), bytex.FromString("3"))
			w.Failed(ErrTooEarly)
			return
		}

		middleware.counter.Add(1)
		// set runtime into request context
		runtime.With(r, middleware.rt)
		// set request and response into context
		transports.WithRequest(r, r)
		transports.WithResponse(r, w)
		// next
		next.Handle(w, r)

		// check hijacked
		if w.Hijacked() {
			middleware.counter.Done()
			return
		}
		// done
		middleware.counter.Done()
	})
}

func (middleware *ProxyMiddleware) Close() {
	middleware.counter.Wait()
}

package runtime

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"net/http"
	"sync"
)

var (
	ErrTooEarly    = errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: service is not ready, try later again")
	ErrUnavailable = errors.Unavailable("fns: server is closed")
)

func Middleware(rt *Runtime) transports.Middleware {
	return &middleware{
		log:     nil,
		rt:      rt,
		counter: sync.WaitGroup{},
	}
}

type middleware struct {
	log     logs.Logger
	rt      *Runtime
	counter sync.WaitGroup
}

func (middle *middleware) Name() string {
	return "runtime"
}

func (middle *middleware) Construct(options transports.MiddlewareOptions) error {
	middle.log = options.Log
	return nil
}

func (middle *middleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
		running, upped := middle.rt.Running()
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

		middle.counter.Add(1)
		// request Id
		requestId := r.Header().Get(bytex.FromString(transports.RequestIdHeaderName))
		if len(requestId) == 0 {
			requestId = uid.Bytes()
			r.Header().Set(bytex.FromString(transports.RequestIdHeaderName), requestId)
		}
		// set runtime into request context
		With(r, middle.rt)
		// set request and response into context
		transports.WithRequest(r, r)
		transports.WithResponse(r, w)
		// next
		next.Handle(w, r)
		// check hijacked
		if w.Hijacked() {
			middle.counter.Done()
			return
		}

		// done
		middle.counter.Done()
	})
}

func (middle *middleware) Close() {
	middle.counter.Wait()
	return
}

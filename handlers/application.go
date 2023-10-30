package handlers

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/transports"
	"sync"
)

type Application struct {
	rt      *runtime.Runtime
	counter sync.WaitGroup
}

func (app *Application) Name() string {
	return "application"
}

func (app *Application) Construct(options transports.MiddlewareOptions) error {

	return nil
}

func (app *Application) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
		running, upped := app.rt.Running()
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
		// header >>>
		// device id
		r.Header()

		// request id

		// header <<<
		app.counter.Add(1)
		// set runtime into request context
		runtime.With(r, app.rt)
		// next
		next.Handle(w, r)
		// check hijacked
		if w.Hijacked() {
			app.counter.Done()
			return
		}
		// header >>>
		// request id

		// latency

		// header <<<
		// done
		app.counter.Done()
	})
}

func (app *Application) Close() {
	app.counter.Wait()
	return
}

package handlers

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/transports"
)

type Application struct {
	rt *runtime.Runtime
}

func (app *Application) Name() string {
	return "application"
}

func (app *Application) Construct(options transports.MiddlewareOptions) error {

	return nil
}

func (app *Application) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
		r.SetUserValue(bytex.FromString(runtime.ContextKey), app.rt)
		next.Handle(w, r)
	})
}

func (app *Application) Close() {
	return
}

package transports

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/logs"
)

type HandlerBuilderOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type HandlerBuilder interface {
	Name() (name string)
	Build(options HandlerBuilder) (handler Handler, err error)
}

type Handler interface {
	Handle(w ResponseWriter, r *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) Handle(w ResponseWriter, r *Request) {
	f(w, r)
}

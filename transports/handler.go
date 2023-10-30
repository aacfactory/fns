package transports

import "github.com/aacfactory/errors"

type Handler interface {
	Handle(w ResponseWriter, r Request)
}

type HandlerFunc func(ResponseWriter, Request)

func (f HandlerFunc) Handle(w ResponseWriter, r Request) {
	f(w, r)
}

type MuxHandler interface {
	Match(method []byte, path []byte, header Header) bool
	Handler
}

func NewMux() *Mux {
	return &Mux{
		handlers: make([]MuxHandler, 0, 1),
	}
}

type Mux struct {
	handlers []MuxHandler
}

func (mux *Mux) Add(handler MuxHandler) {
	mux.handlers = append(mux.handlers, handler)
}

func (mux *Mux) Handle(w ResponseWriter, r Request) {
	for _, handler := range mux.handlers {
		matched := handler.Match(r.Method(), r.Path(), r.Header())
		if matched {
			handler.Handle(w, r)
			return
		}
	}
	w.Failed(errors.NotFound("fns: not found").WithMeta("handler", "mux"))
}

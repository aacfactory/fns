package handlers

import (
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/transports"
)

type Handler struct {
	rt *runtime.Runtime
}

func (h *Handler) Handle(w transports.ResponseWriter, r transports.Request) {
	//TODO implement me
	panic("implement me")
}

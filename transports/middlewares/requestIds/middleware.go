package requestIds

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/transports"
)

func Middleware() transports.Middleware {
	return &middleware{}
}

type middleware struct {
}

func (m *middleware) Name() string {
	return "requestIds"
}

func (m *middleware) Construct(_ transports.MiddlewareOptions) error {
	return nil
}

func (m *middleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
		requestId := r.Header().Get(bytex.FromString(transports.RequestIdHeaderName))
		if len(requestId) == 0 {
			requestId = uid.Bytes()
			r.Header().Set(bytex.FromString(transports.RequestIdHeaderName), requestId)
		}

		next.Handle(w, r)
		// check hijacked
		if w.Hijacked() {
			return
		}
		w.Header().Set(bytex.FromString(transports.RequestIdHeaderName), requestId)
	})
}

func (m *middleware) Close() {
	return
}

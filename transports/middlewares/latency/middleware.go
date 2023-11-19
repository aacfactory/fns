package latency

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"time"
)

func Middleware() transports.Middleware {
	return &middleware{}
}

type Config struct {
	Enabled bool `json:"enabled"`
}

type middleware struct {
	enabled bool
}

func (m *middleware) Name() string {
	return "latency"
}

func (m *middleware) Construct(options transports.MiddlewareOptions) error {
	config := Config{}
	err := options.Config.As(&config)
	if err != nil {
		err = errors.Warning("fns: construct latency middleware failed").WithCause(err)
		return err
	}
	m.enabled = config.Enabled
	return nil
}

func (m *middleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
		beg := time.Time{}
		if m.enabled {
			beg = time.Now()
		}
		next.Handle(w, r)
		if w.Hijacked() {
			return
		}
		latency := time.Now().Sub(beg)
		w.Header().Set(transports.HandleLatencyHeaderName, bytex.FromString(latency.String()))
	})
}

func (m *middleware) Close() {
}

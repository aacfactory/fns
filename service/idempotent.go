package service

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/shareds"
	shareds2 "github.com/aacfactory/fns/shareds"
	transports2 "github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"strings"
	"time"
)

const (
	idempotentMiddlewareName         = "idempotent"
	idempotentMiddlewareTicketPrefix = "fns/middlewares/idempotent"
)

func IdempotentMiddleware() TransportMiddleware {
	return &idempotentMiddleware{}
}

type idempotentMiddlewareConfig struct {
	TicketTTL string `json:"ticketTTL"`
}

type idempotentMiddleware struct {
	log       logs.Logger
	ticketTTL time.Duration
	tickets   shareds.Caches
}

func (middleware *idempotentMiddleware) Name() (name string) {
	name = idempotentMiddlewareName
	return
}

func (middleware *idempotentMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.log = options.Log
	config := idempotentMiddlewareConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: idempotent middleware build failed").WithCause(configErr)
		return
	}
	config.TicketTTL = strings.TrimSpace(config.TicketTTL)
	if config.TicketTTL == "" {
		config.TicketTTL = "120s"
	}
	ticketTTL, parseTicketTTLErr := time.ParseDuration(config.TicketTTL)
	if parseTicketTTLErr != nil {
		parseTicketTTLErr = errors.Warning("fns: ticketTTL must be time.Duration format").WithCause(parseTicketTTLErr)
		err = errors.Warning("fns: idempotent middleware build failed").WithCause(parseTicketTTLErr)
		return
	}
	if ticketTTL < 1 {
		ticketTTL = 2 * time.Minute
	}
	middleware.ticketTTL = ticketTTL
	middleware.tickets = options.Runtime.Shared().Caches()
	return
}

func (middleware *idempotentMiddleware) Handler(next transports2.Handler) transports2.Handler {
	return transports2.HandlerFunc(func(w transports2.ResponseWriter, r *transports2.Request) {
		if r.Header().Get(httpRequestInternalHeader) != "" {
			next.Handle(w, r)
			return
		}
		rh := getOrMakeRequestHash(r.Header(), r.Path(), r.Body())
		key := bytes.Join([][]byte{bytex.FromString(idempotentMiddlewareTicketPrefix), rh}, slashBytes)
		prev, ok := middleware.tickets.Set(r.Context(), key, []byte{'1'}, middleware.ticketTTL, shareds2.SystemScope())
		if !ok {
			w.Failed(ErrLockedRequest.WithCause(errors.Warning("fns: save request idempotent ticket failed")))
			return
		}
		if len(prev) > 0 {
			w.Failed(ErrLockedRequest)
			return
		}
		next.Handle(w, r)
		middleware.tickets.Remove(r.Context(), key, shareds2.SystemScope())
	})
}

func (middleware *idempotentMiddleware) Close() (err error) {
	return
}

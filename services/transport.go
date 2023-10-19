package services

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/transports"
	"time"
)

type Transport struct {
	transport   transports.Transport
	middlewares *transportMiddlewares
	handlers    *transportHandlers
	port        int
}

func (tr *Transport) Port() int {
	return tr.port
}

func (tr *Transport) Listen(ctx context.Context) (err error) {
	err = tr.middlewares.Build()
	if err != nil {
		err = errors.Warning("fns: transport listen failed").WithCause(err)
		return
	}
	err = tr.handlers.Build()
	if err != nil {
		err = errors.Warning("fns: transport listen failed").WithCause(err)
		return
	}
	errCh := make(chan error, 2)
	go func(srv transports.Transport, ch chan error) {
		listenErr := srv.ListenAndServe()
		if listenErr != nil {
			ch <- errors.Warning("fns: transport listen failed").WithCause(listenErr)
			close(ch)
		}
	}(tr.transport, errCh)
	timeout := 2 * time.Second
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		if remain := deadline.Sub(time.Now()); remain < timeout {
			timeout = remain
		}
	}
	select {
	case err = <-errCh:
		break
	case <-time.After(timeout):
		break
	}
	return
}

func (tr *Transport) Close() (err error) {
	errs := errors.MakeErrors()
	err = tr.middlewares.Close()
	if err != nil {
		errs.Append(err)
	}
	err = tr.handlers.Close()
	if err != nil {
		errs.Append(err)
	}
	err = tr.transport.Close()
	if err != nil {
		errs.Append(err)
	}
	err = errs.Error()
	return
}

func (tr *Transport) services() (v []Service) {
	v = make([]Service, 0, 1)
	for _, middleware := range tr.middlewares.middlewares {
		servicesSupplier, ok := middleware.(Supplier)
		if ok && servicesSupplier.Services() != nil {
			v = append(v, servicesSupplier.Services()...)
		}
	}
	for _, handler := range tr.handlers.handlers {
		servicesSupplier, ok := handler.(Supplier)
		if ok && servicesSupplier.Services() != nil {
			v = append(v, servicesSupplier.Services()...)
		}
	}
	return
}

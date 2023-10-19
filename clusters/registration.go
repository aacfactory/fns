package clusters

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/commons/window"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/workers"
	"sync"
	"sync/atomic"
	"time"
)

type Registration struct {
	hostId   string
	id       string
	version  versions.Version
	address  string
	name     string
	document *documents.Document
	devMode  bool // todo remove
	client   transports.Client
	signer   signatures.Signature
	worker   workers.Workers
	timeout  time.Duration
	pool     sync.Pool
	closed   *atomic.Bool
	errs     *window.Times
}

func (registration *Registration) Key() (key string) {
	key = registration.id
	return
}

func (registration *Registration) Name() (name string) {
	name = registration.name
	return
}

func (registration *Registration) Internal() (ok bool) {
	ok = true
	return
}

func (registration *Registration) Document() (document *documents.Document) {
	document = registration.document
	return
}

func (registration *Registration) Request(ctx context.Context, r services.Request) (future services.Future) {
	promise, fr := services.NewFuture()
	task := registration.acquire()
	task.begin(r, promise)
	if !registration.worker.Dispatch(ctx, task) {
		promise.Failed(services.ErrServiceOverload)
		registration.release(task)
	}
	future = fr
	return
}

func (registration *Registration) RequestSync(ctx context.Context, r services.Request) (result services.FutureResult, err errors.CodeError) {
	fr := registration.Request(ctx, r)
	result, err = fr.Get(ctx)
	return
}

func (registration *Registration) Close() {
	registration.closed.Store(true)
	registration.client.Close()
}

func (registration *Registration) acquire() (task *registrationTask) {
	v := registration.pool.Get()
	if v != nil {
		task = v.(*registrationTask)
		return
	}
	task = newRegistrationTask(registration, registration.timeout, func(task *registrationTask) {
		registration.release(task)
	})
	return
}

func (registration *Registration) release(task *registrationTask) {
	task.end()
	registration.pool.Put(task)
	return
}

type RegistrationList []*Registration

func (list RegistrationList) Len() int {
	return len(list)
}

func (list RegistrationList) Less(i, j int) bool {
	return list[i].version.LessThan(list[j].version)
}

func (list RegistrationList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
	return
}

func (list RegistrationList) MinVersion() (r *Registration) {
	size := len(list)
	if size == 0 {
		return
	}
	r = list[0]
	return
}

func (list RegistrationList) MaxVersion() (r *Registration) {
	size := len(list)
	if size == 0 {
		return
	}
	r = list[size-1]
	return
}

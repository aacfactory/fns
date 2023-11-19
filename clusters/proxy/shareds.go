package proxy

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
)

var (
	sharedHandlerPath = append(handlerPathPrefix, []byte("/clusters/shared")...)
	sharedHeader      = []byte("X-Fns-Shared")
)

func NewShared(client transports.Client, signature signatures.Signature) shareds.Shared {
	return &Shared{
		lockers: &Lockers{
			client:    client,
			signature: signature,
		},
		store: &Store{
			client:    client,
			signature: signature,
		},
	}
}

type Shared struct {
	lockers shareds.Lockers
	store   shareds.Store
}

func (shared *Shared) Construct(_ shareds.Options) (err error) {
	return
}

func (shared *Shared) Lockers() (lockers shareds.Lockers) {
	lockers = shared.lockers
	return
}

func (shared *Shared) Store() (store shareds.Store) {
	store = shared.store
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewSharedHandler(shared shareds.Shared) transports.Handler {
	return &SharedHandler{
		lockers: NewSharedLockersHandler(shared.Lockers()),
		store:   NewSharedStoreHandler(shared.Store()),
	}
}

type SharedHandler struct {
	lockers transports.Handler
	store   transports.Handler
}

func (handler *SharedHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	kind := r.Header().Get(sharedHeader)
	if bytes.Equal(kind, sharedHeaderLockersValue) {
		handler.lockers.Handle(w, r)
	} else if bytes.Equal(kind, sharedHeaderStoreValue) {
		handler.store.Handle(w, r)
	} else {
		w.Failed(errors.Warning("fns: X-Fns-Shared is required"))
	}
	return
}

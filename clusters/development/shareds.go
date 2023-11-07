package development

import (
	"bytes"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
)

func NewShared(dialer transports.Dialer, address []byte, signature signatures.Signature) shareds.Shared {
	return &Shared{
		lockers: &Lockers{
			address:   address,
			dialer:    dialer,
			signature: signature,
		},
		store: &Store{
			address:   address,
			dialer:    dialer,
			signature: signature,
		},
	}
}

type Shared struct {
	lockers shareds.Lockers
	store   shareds.Store
}

func (shared *Shared) Construct(options shareds.Options) (err error) {
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

var (
	shardHandlePathPrefix = []byte("/development/shared/")
)

func NewSharedHandler(signature signatures.Signature, shared shareds.Shared) transports.Handler {
	return &SharedHandler{
		signature: signature,
		lockers:   NewSharedLockersHandler(shared.Lockers(), signature),
		store:     NewSharedStoreHandler(shared.Store(), signature),
	}
}

type SharedHandler struct {
	signature signatures.Signature
	lockers   transports.Handler
	store     transports.Handler
}

func (handler *SharedHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	path := r.Path()
	if bytes.Equal(path, shardHandleLockersPath) {
		handler.lockers.Handle(w, r)
	} else if bytes.Equal(path, shardHandleStorePath) {
		handler.store.Handle(w, r)
	} else {
		w.Failed(ErrInvalidPath.WithMeta("path", string(path)))
	}
	return
}

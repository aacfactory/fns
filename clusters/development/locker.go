package development

import (
	"context"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"time"
)

type Locker struct {
}

func (locker *Locker) Lock(ctx context.Context) (err error) {
	//TODO implement me
	panic("implement me")
}

func (locker *Locker) Unlock(ctx context.Context) (err error) {
	//TODO implement me
	panic("implement me")
}

type Lockers struct {
	address   []byte
	dialer    transports.Dialer
	signature signatures.Signature
}

func (lockers *Lockers) Acquire(ctx context.Context, key []byte, ttl time.Duration) (locker shareds.Locker, err error) {
	//TODO implement me
	panic("implement me")
}

// +-------------------------------------------------------------------------------------------------------------------+

var (
	shardHandleLockersPath = []byte("/development/shared/lockers")
)

func NewSharedLockersHandler(lockers shareds.Lockers, signature signatures.Signature) transports.Handler {
	return &SharedLockersHandler{
		lockers:   lockers,
		signature: signature,
	}
}

type SharedLockersHandler struct {
	lockers   shareds.Lockers
	signature signatures.Signature
}

func (handler *SharedLockersHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	//TODO implement me
	panic("implement me")
}

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

type Store struct {
	address   []byte
	dialer    transports.Dialer
	signature signatures.Signature
}

func (store *Store) Keys(ctx context.Context, prefix []byte) (keys [][]byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (store *Store) Get(ctx context.Context, key []byte) (value []byte, has bool, err error) {
	//TODO implement me
	panic("implement me")
}

func (store *Store) Set(ctx context.Context, key []byte, value []byte) (err error) {
	//TODO implement me
	panic("implement me")
}

func (store *Store) SetWithTTL(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error) {
	//TODO implement me
	panic("implement me")
}

func (store *Store) Incr(ctx context.Context, key []byte, delta int64) (v int64, err error) {
	//TODO implement me
	panic("implement me")
}

func (store *Store) ExpireKey(ctx context.Context, key []byte, ttl time.Duration) (err error) {
	//TODO implement me
	panic("implement me")
}

func (store *Store) Remove(ctx context.Context, key []byte) (err error) {
	//TODO implement me
	panic("implement me")
}

func (store *Store) Close() {
	//TODO implement me
	panic("implement me")
}

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

func NewSharedHandler(signature signatures.Signature) transports.Handler {
	return &SharedHandler{
		signature: signature,
	}
}

type SharedHandler struct {
	signature signatures.Signature
}

func (handler *SharedHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	//TODO implement me
	panic("implement me")
}

package development

import (
	"context"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"time"
)

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

// +-------------------------------------------------------------------------------------------------------------------+

var (
	shardHandleStorePath = []byte("/development/shared/store")
)

func NewSharedStoreHandler(store shareds.Store, signature signatures.Signature) transports.Handler {
	return &SharedStoreHandler{
		store:     store,
		signature: signature,
	}
}

type SharedStoreHandler struct {
	store     shareds.Store
	signature signatures.Signature
}

func (handler *SharedStoreHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	//TODO implement me
	panic("implement me")
}

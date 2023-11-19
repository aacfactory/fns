package caches

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/logs"
	"time"
)

type KeyParam interface {
	CacheKey(ctx context.Context) (key []byte, err error)
}

type Store interface {
	services.Component
	Get(ctx context.Context, key []byte) (value []byte, has bool, err error)
	Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error)
	Remove(ctx context.Context, key []byte) (err error)
}

type defaultStore struct {
	log    logs.Logger
	prefix []byte
}

func (store *defaultStore) Name() (name string) {
	return "default"
}

func (store *defaultStore) Construct(options services.Options) (err error) {
	store.log = options.Log
	store.prefix = []byte("fns/caches/")
	return
}

func (store *defaultStore) Shutdown(_ context.Context) {
	return
}

func (store *defaultStore) Get(ctx context.Context, key []byte) (value []byte, has bool, err error) {
	st := runtime.SharedStore(ctx)
	value, has, err = st.Get(ctx, append(store.prefix, key...))
	if err != nil {
		err = errors.Warning("fns: get cache failed").WithMeta("key", string(key)).WithCause(err)
		return
	}
	return
}

func (store *defaultStore) Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error) {
	st := runtime.SharedStore(ctx)
	err = st.SetWithTTL(ctx, append(store.prefix, key...), value, ttl)
	if err != nil {
		err = errors.Warning("fns: set cache failed").WithMeta("key", string(key)).WithCause(err)
		return
	}
	return
}

func (store *defaultStore) Remove(ctx context.Context, key []byte) (err error) {
	st := runtime.SharedStore(ctx)
	err = st.Remove(ctx, append(store.prefix, key...))
	if err != nil {
		err = errors.Warning("fns: remove cache failed").WithMeta("key", string(key)).WithCause(err)
		return
	}
	return
}

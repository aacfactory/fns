package cachecontrol

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/runtime"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key []byte) (value []byte, has bool, err error)
	Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error)
	Close()
}

var (
	cacheKeyPrefix = []byte("$.fns:cachecontrol:")
)

type DefaultCache struct{}

func (cache *DefaultCache) Get(ctx context.Context, key []byte) (value []byte, has bool, err error) {
	store := runtime.SharedStore(ctx)
	value, has, err = store.Get(ctx, append(cacheKeyPrefix, key...))
	if err != nil {
		err = errors.Warning("fns: cache control store get failed").WithCause(err)
		return
	}
	return
}

func (cache *DefaultCache) Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error) {
	store := runtime.SharedStore(ctx)
	err = store.SetWithTTL(ctx, append(cacheKeyPrefix, key...), value, ttl)
	if err != nil {
		err = errors.Warning("fns: cache control store set failed").WithCause(err)
		return
	}
	return
}

func (cache *DefaultCache) Close() {
}

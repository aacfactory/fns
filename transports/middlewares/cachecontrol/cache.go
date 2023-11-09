package cachecontrol

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/cespare/xxhash/v2"
	"strconv"
	"time"
)

// todo
// middleware
// shared store 存 etag
// inbound
// internal的话，直接跳过（由services cache处理）
// if non match 存在，则去匹配，
// key=device id + token + body
// value=etag
// 当匹配时，返回204
// 不匹配时，往store里绑定request cache id的cache item{hash，etag，max-age}
// outbound
// 成功时，hash result成etag，往store里设
// handler后，
// 取cache item，如果有变动，则store里放etag
// 如果非internal，则response header里放etag等等，
// 删除store里的cache item
// 如果是proxy模式，也是一样，此时backed 就不要上middleware，但是设置cache的函数是可以用的，因为store

type Cache interface {
	Get(ctx context.Context, key []byte) (value []byte, has bool, err error)
	Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error)
	Close()
}

var (
	cacheKeyPrefix = []byte("fns:cachecontrol:")
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

var (
	cacheContextKey = []byte("@fns:middleware:cachecontrol:cache")
)

func withCache(ctx transports.Request, cache Cache) {
	ctx.SetUserValue(cacheContextKey, cache)
}

func tryLoadCache(ctx context.Context) (cache Cache, has bool) {
	v := ctx.Value(cacheContextKey)
	if v == nil {
		return
	}
	cache, has = v.(Cache)
	return
}

type MakeOptions struct {
	mustRevalidate  bool
	proxyRevalidate bool
	public          bool
	maxAge          int
}

type MakeOption func(option *MakeOptions)

func MustRevalidate() MakeOption {
	return func(option *MakeOptions) {
		option.mustRevalidate = true
	}
}

func ProxyRevalidate() MakeOption {
	return func(option *MakeOptions) {
		option.proxyRevalidate = true
	}
}

func Private() MakeOption {
	return func(option *MakeOptions) {
		option.public = false
	}
}

func Public() MakeOption {
	return func(option *MakeOptions) {
		option.public = true
	}
}

func MaxAge(age int) MakeOption {
	return func(option *MakeOptions) {
		if age < 0 {
			age = 0
		}
		option.maxAge = age
	}
}

var (
	entryCacheKeyPrefix = []byte("@fns:cachecontrol:entry:")
)

type Entry struct {
	MustRevalidate  bool
	ProxyRevalidate bool
	Public          bool
	MaxAge          int
	Key             []byte
	ETag            []byte
}

func Make(ctx context.Context, body interface{}, options ...MakeOption) (err error) {
	if body == nil {
		return
	}
	header, hasHeader := transports.TryLoadRequestHeader(ctx)
	if !hasHeader {
		err = errors.Warning("fns: make cache control failed").WithCause(fmt.Errorf("there is no transport request in context"))
		return
	}
	requestId := header.Get(bytex.FromString(transports.RequestIdHeaderName))
	if len(requestId) == 0 {
		err = errors.Warning("fns: make cache control failed").WithCause(fmt.Errorf("there is no X-Fns-Request-Id in transport request"))
		return
	}
	store := runtime.SharedStore(ctx)
	ek := append(entryCacheKeyPrefix, requestId...)
	ep, has, getErr := store.Get(ctx, ek)
	if getErr != nil {
		err = errors.Warning("fns: make cache control failed").WithCause(getErr)
		return
	}
	if !has {
		err = errors.Warning("fns: make cache control failed").WithCause(fmt.Errorf("cache control was not enabled"))
		return
	}
	entry := Entry{}
	decodeErr := json.Unmarshal(ep, &entry)
	if decodeErr != nil {
		err = errors.Warning("fns: make cache control failed").WithCause(decodeErr)
		return
	}
	p, encodeErr := json.Marshal(body)
	if encodeErr != nil {
		err = errors.Warning("fns: make cache control failed").WithCause(encodeErr)
		return
	}
	entry.ETag = bytex.FromString(strconv.FormatUint(xxhash.Sum64(p), 16))
	opt := MakeOptions{}
	for _, option := range options {
		option(&opt)
	}
	entry.MustRevalidate = opt.mustRevalidate
	entry.ProxyRevalidate = opt.proxyRevalidate
	entry.Public = opt.public
	entry.MaxAge = opt.maxAge

	ep, encodeErr = json.Marshal(entry)
	if encodeErr != nil {
		err = errors.Warning("fns: make cache control failed").WithCause(encodeErr)
		return
	}

	setErr := store.SetWithTTL(ctx, ek, ep, 10*time.Second)
	if setErr != nil {
		err = errors.Warning("fns: make cache control failed").WithCause(setErr)
		return
	}

	return
}

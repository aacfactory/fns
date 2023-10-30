package context

import (
	"context"
	"sync"
	"unsafe"
)

var (
	pool = sync.Pool{}
)

func Acquire(ctx context.Context) Context {
	cached := pool.Get()
	if cached == nil {
		return &context_{
			Context: ctx,
			entries: make(Entries, 0, 1),
		}
	}
	v := cached.(*context_)
	v.Context = ctx
	return v
}

func Release(ctx context.Context) {
	v, ok := ctx.(*context_)
	if ok {
		v.Context = nil
		v.entries.Reset()
		pool.Put(v)
	}
}

func WithValue(parent context.Context, key []byte, val any) Context {
	ctx, ok := parent.(Context)
	if ok {
		ctx.SetUserValue(key, val)
		return ctx
	}
	ctx = &context_{
		Context: ctx,
		entries: make(Entries, 0, 1),
	}
	ctx.SetUserValue(key, val)
	return ctx
}

type Context interface {
	context.Context
	UserValue(key []byte) any
	SetUserValue(key []byte, val any)
	UserValues(fn func(key []byte, val any))
}

type context_ struct {
	context.Context
	entries Entries
}

func (c *context_) UserValue(key []byte) any {
	v := c.entries.Get(key)
	if v != nil {
		return v
	}
	parent, ok := c.Context.(Context)
	if ok {
		return parent.UserValue(key)
	}
	return nil
}

func (c *context_) SetUserValue(key []byte, val any) {
	c.entries.Set(key, val)
}

func (c *context_) UserValues(fn func(key []byte, val any)) {
	parent, ok := c.Context.(Context)
	if ok {
		parent.UserValues(fn)
	}
	c.entries.Foreach(fn)
}

func (c *context_) Value(key any) any {
	switch k := key.(type) {
	case []byte:
		v := c.entries.Get(k)
		if v == nil {
			return c.Context.Value(key)
		}
		return v
	case string:
		v := c.entries.Get(unsafe.Slice(unsafe.StringData(k), len(k)))
		if v == nil {
			return c.Context.Value(key)
		}
		return v
	default:
		break
	}
	return c.Context.Value(key)
}

package context

import (
	"context"
	"unsafe"
)

func Wrap(ctx context.Context) Context {
	v, ok := ctx.(Context)
	if ok {
		return v
	}
	return &context_{
		Context: ctx,
		entries: make(Entries, 0, 1),
	}
}

func Fork(parent Context) Context {
	return &context_{
		Context: parent,
		entries: make(Entries, 0, 1),
	}
}

func WithValue(parent context.Context, key []byte, val any) Context {
	ctx := Wrap(parent)
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
	return c.entries.Get(key)
}

func (c *context_) SetUserValue(key []byte, val any) {
	c.entries.Set(key, val)
}

func (c *context_) UserValues(fn func(key []byte, val any)) {
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

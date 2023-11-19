package context

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/scanner"
	"time"
)

var (
	Canceled         = context.Canceled
	DeadlineExceeded = context.DeadlineExceeded
)

func Wrap(ctx context.Context) Context {
	v, ok := ctx.(Context)
	if ok {
		return v
	}
	return &context_{
		Context: ctx,
		entries: make(Entries, 0, 1),
		locals:  make(Entries, 0, 1),
	}
}

func TODO() Context {
	return Wrap(context.TODO())
}

func WithValue(parent context.Context, key []byte, val any) Context {
	ctx, ok := parent.(Context)
	if ok {
		ctx.SetLocalValue(key, val)
		return ctx
	}
	ctx = &context_{
		Context: ctx,
		entries: make(Entries, 0, 1),
		locals:  make(Entries, 0, 1),
	}
	ctx.SetLocalValue(key, val)
	return ctx
}

func ScanValue(ctx context.Context, key any, val any) (has bool, err error) {
	v := ctx.Value(key)
	if v == nil {
		return
	}
	s := scanner.New(v)
	err = s.Scan(val)
	if err != nil {
		err = errors.Warning("fns: scan context value failed").WithCause(err)
		return
	}
	has = true
	return
}

type CancelFunc context.CancelFunc

func WithCancel(parent Context) (Context, CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithTimeout(parent Context, ttl time.Duration) (Context, CancelFunc) {
	ctx, cancel := context.WithTimeout(parent, ttl)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithTimeoutCause(parent Context, ttl time.Duration, cause error) (Context, CancelFunc) {
	ctx, cancel := context.WithTimeoutCause(parent, ttl, cause)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc) {
	ctx, cancel := context.WithDeadline(parent, deadline)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithDeadlineCause(parent Context, deadline time.Time, cause error) (Context, CancelFunc) {
	ctx, cancel := context.WithDeadlineCause(parent, deadline, cause)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithoutCancel(parent Context) Context {
	return Wrap(context.WithoutCancel(parent))
}

func AfterFunc(ctx Context, f func()) (stop func() bool) {
	stop = context.AfterFunc(ctx, f)
	return
}

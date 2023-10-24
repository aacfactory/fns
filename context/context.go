package context

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/logger"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/logs"
	"time"
)

type CancelFunc context.CancelFunc
type CancelCauseFunc context.CancelCauseFunc

func Wrap(ctx context.Context) Context {
	return &context_{
		ctx,
	}
}

func WithValue(parent context.Context, key interface{}, value interface{}) Context {
	return Wrap(context.WithValue(parent, key, value))
}

func WithCancel(parent context.Context) (Context, CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithCancelCause(parent context.Context) (Context, CancelCauseFunc) {
	ctx, cancel := context.WithCancelCause(parent)
	return Wrap(ctx), CancelCauseFunc(cancel)
}

func WithoutCancel(parent context.Context) Context {
	return Wrap(context.WithoutCancel(parent))
}

func WithTimeout(parent context.Context, timeout time.Duration) (Context, CancelFunc) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithTimeoutCause(parent context.Context, timeout time.Duration, cause error) (Context, CancelFunc) {
	ctx, cancel := context.WithTimeoutCause(parent, timeout, cause)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithDeadline(parent context.Context, d time.Time) (Context, CancelFunc) {
	ctx, cancel := context.WithDeadline(parent, d)
	return Wrap(ctx), CancelFunc(cancel)
}

func WithDeadlineCause(parent context.Context, d time.Time, cause error) (Context, CancelFunc) {
	ctx, cancel := context.WithDeadlineCause(parent, d, cause)
	return Wrap(ctx), CancelFunc(cancel)
}

func Cause(parent context.Context) error {
	return context.Cause(parent)
}

func AfterFunc(ctx Context, f func()) (stop func() bool) {
	stop = context.AfterFunc(ctx, f)
	return
}

type Context interface {
	context.Context
	Log() logs.Logger
	Request() services.Request
	Runtime() *runtime.Runtime
}

type context_ struct {
	context.Context
}

func (c *context_) Log() logs.Logger {
	return logger.Log(c)
}

func (c *context_) Request() services.Request {
	v := c.Value(services.ContextRequestKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no request in context")))
		return nil
	}
	r, ok := v.(services.Request)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: request in context is not github.com/aacfactory/fns/services.Request")))
		return nil
	}
	return r
}

func (c *context_) Components() services.Components {
	v := c.Value(services.ContextComponentsKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no components in context")))
		return nil
	}
	r, ok := v.(services.Components)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: components in context is not github.com/aacfactory/fns/services.Components")))
		return nil
	}
	return r
}

func (c *context_) Runtime() *runtime.Runtime {
	return runtime.Load(c)
}

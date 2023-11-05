package barriers

import (
	"context"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/logs"
	"golang.org/x/sync/singleflight"
)

type ErrorReporter func(ctx context.Context, cause error)

type Result struct {
	objects.Scanner
}

type Options struct {
	Log    logs.Logger
	Config configures.Config
}

type Barrier interface {
	Construct(options Options) (err error)
	Do(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (result Result, err error)
	Forget(ctx context.Context, key []byte)
}

func New() (b Barrier) {
	b = &barrier{}
	return
}

// Barrier
// @barrier
// todo 当@authorization 存在时，则key增加user，不存在时，不加user
type barrier struct {
	group *singleflight.Group
}

func (b *barrier) Construct(options Options) (err error) {
	b.group = new(singleflight.Group)
	return
}

func (b *barrier) Do(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (result Result, err error) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	r, doErr, _ := b.group.Do(bytex.ToString(key), func() (r interface{}, err error) {
		r, err = fn()
		return
	})
	if doErr != nil {
		err = errors.Map(doErr)
		return
	}
	result = Result{
		objects.NewScanner(r),
	}
	return
}

func (b *barrier) Forget(ctx context.Context, key []byte) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	b.group.Forget(bytex.ToString(key))
}

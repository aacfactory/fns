package barriers

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/context"
	"golang.org/x/sync/singleflight"
)

type Result interface {
	scanner.Scanner
}

type Barrier interface {
	Do(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (result Result, err error)
	Forget(ctx context.Context, key []byte)
}

func New() (b Barrier) {
	b = &barrier{
		group: new(singleflight.Group),
	}
	return
}

// Barrier
// @barrier
// 当@authorization 存在时，则key为 services.HashRequest(r, services.HashRequestWithToken())
type barrier struct {
	group *singleflight.Group
}

func (b *barrier) Do(_ context.Context, key []byte, fn func() (result interface{}, err error)) (r Result, err error) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	v, doErr, _ := b.group.Do(bytex.ToString(key), func() (v interface{}, err error) {
		v, err = fn()
		return
	})
	if doErr != nil {
		err = errors.Map(doErr)
		return
	}
	r = scanner.New(v)
	return
}

func (b *barrier) Forget(_ context.Context, key []byte) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	b.group.Forget(bytex.ToString(key))
}

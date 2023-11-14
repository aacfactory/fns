package futures

import (
	"context"
	"github.com/aacfactory/errors"
	"sync"
)

type Promise interface {
	Succeed(v interface{})
	Failed(err error)
}

type Future interface {
	Get(ctx context.Context) (result Result, err error)
}

var (
	pool = sync.Pool{
		New: func() any {
			return make(chan value, 1)
		},
	}
)

func New() (p Promise, f Future) {
	ch := pool.Get().(chan value)
	p = promise{
		ch: ch,
	}
	f = future{
		ch: ch,
	}
	return
}

type value struct {
	val any
	err error
}

type promise struct {
	ch chan value
}

func (p promise) Succeed(v interface{}) {
	p.ch <- value{
		val: v,
	}
}

func (p promise) Failed(err error) {
	if err == nil {
		err = errors.Warning("fns: empty failed result").WithMeta("fns", "futures")
	}
	p.ch <- value{
		err: err,
	}
}

type future struct {
	ch chan value
}

func (f future) Get(ctx context.Context) (r Result, err error) {
	select {
	case <-ctx.Done():
		err = errors.Timeout("fns: get result value from future timeout").WithMeta("fns", "futures")
		break
	case data, ok := <-f.ch:
		pool.Put(f.ch)
		if !ok {
			err = errors.Warning("fns: future was closed").WithMeta("fns", "futures")
			break
		}
		if data.err != nil {
			err = data.err
			break
		}
		r = result{
			value: data.val,
		}
		break
	}
	return
}

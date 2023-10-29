package futures

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/json"
	"sync"
)

type Promise interface {
	Succeed(v interface{})
	Failed(err error)
}

type Future interface {
	Get(ctx context.Context) (result Result, err error)
}

type Result interface {
	json.Marshaler
	Exist() (ok bool)
	Scan(v interface{}) (err error)
}

var (
	pool = sync.Pool{
		New: func() any {
			return make(chan result, 1)
		},
	}
)

func New(callback ...func()) (p Promise, f Future) {
	ch := pool.Get().(chan result)
	p = promise{
		ch: ch,
	}
	f = future{
		ch:        ch,
		callbacks: callback,
	}
	return
}

type result struct {
	val any
	err error
}

type promise struct {
	ch chan result
}

func (p promise) Succeed(v interface{}) {
	p.ch <- result{
		val: v,
	}
}

func (p promise) Failed(err error) {
	if err == nil {
		err = errors.Warning("fns: empty failed result").WithMeta("fns", "futures")
	}
	p.ch <- result{
		err: err,
	}
}

type future struct {
	ch        chan result
	callbacks []func()
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
		r = objects.NewScanner(data.val)
		break
	}

	for _, callback := range f.callbacks {
		callback()
	}
	return
}

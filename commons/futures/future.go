package futures

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/json"
)

type Promise interface {
	Succeed(v interface{})
	Failed(err error)
}

type Result interface {
	json.Marshaler
	Exist() (ok bool)
	Scan(v interface{}) (err error)
}

type Future interface {
	Get(ctx context.Context) (result Result, err error)
}

func New() (p Promise, f Future) {
	fp := pipe{
		ch: make(chan result, 1),
	}
	p = fp
	f = fp
	return
}

type result struct {
	val any
	err error
}

type pipe struct {
	ch chan result
}

func (p pipe) Succeed(v interface{}) {
	p.ch <- result{
		val: v,
	}
	close(p.ch)
}

func (p pipe) Failed(err error) {
	if err == nil {
		err = errors.Warning("fns: empty failed result").WithMeta("fns", "future")
	}
	p.ch <- result{
		err: err,
	}
	close(p.ch)
}

func (p pipe) Get(ctx context.Context) (r Result, err error) {
	select {
	case <-ctx.Done():
		err = errors.Timeout("fns: get result value timeout").WithMeta("fns", "future")
		return
	case data, ok := <-p.ch:
		if !ok {
			err = errors.Warning("fns: future was closed").WithMeta("fns", "future")
			return
		}
		if data.err != nil {
			err = data.err
			return
		}
		r = objects.NewScanner(data.val)
		return
	}
}

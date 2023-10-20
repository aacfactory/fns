package futures

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/json"
	"sync/atomic"
)

type Promise interface {
	Succeed(v interface{})
	Failed(err error)
	Close()
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
	fp := &pipe{
		ch:     make(chan interface{}, 1),
		closed: &atomic.Bool{},
	}
	p = fp
	f = fp
	return
}

type result struct {
	objects.Scanner
}

type pipe struct {
	ch     chan interface{}
	closed *atomic.Bool
}

func (p *pipe) Close() {
	if p.closed.Load() {
		return
	}
	p.closed.Store(true)
	close(p.ch)
}

func (p *pipe) Succeed(v interface{}) {
	p.ch <- v
	p.Close()
}

func (p *pipe) Failed(err error) {
	if err == nil {
		err = errors.Warning("fns: empty failed result").WithMeta("fns", "future")
	}
	p.ch <- err
	p.Close()
}

func (p *pipe) Get(ctx context.Context) (r Result, err error) {
	select {
	case <-ctx.Done():
		err = errors.Timeout("fns: get result value timeout").WithMeta("fns", "future")
		return
	case data, ok := <-p.ch:
		if !ok {
			err = errors.Warning("fns: future was closed").WithMeta("fns", "future")
			return
		}
		if data == nil {
			r = objects.NewScanner(nil)
			return
		}
		switch data.(type) {
		case errors.CodeError:
			err = data.(errors.CodeError)
			return
		case error:
			err = errors.Map(data.(error))
			return
		default:
			r = objects.NewScanner(data)
			//r = result(objects.NewScanner(data))
		}
		return
	}
}

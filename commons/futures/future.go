/*
 * Copyright 2023 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package futures

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/objects"
	"sync"
)

type Promise interface {
	Succeed(v interface{})
	Failed(err error)
}

type Result interface {
	objects.Object
}

type Future interface {
	Get(ctx context.Context) (result Result, err error)
}

var (
	pool = sync.Pool{
		New: func() any {
			return &sch{
				locker: new(sync.Mutex),
				ch:     make(chan value, 1),
				closed: false,
			}
		},
	}
)

type sch struct {
	locker sync.Locker
	ch     chan value
	closed bool
}

func (s *sch) send(v value) {
	s.locker.Lock()
	if s.closed {
		s.locker.Unlock()
		return
	}
	s.ch <- v
	s.locker.Unlock()
}

func (s *sch) destroy() {
	s.locker.Lock()
	s.closed = true
	close(s.ch)
	if len(s.ch) > 0 {
		<-s.ch
	}
	s.locker.Unlock()
}

func (s *sch) get() <-chan value {
	return s.ch
}

func New() (p Promise, f Future) {
	ch := pool.Get().(*sch)
	p = promise{
		ch: ch,
	}
	f = future{
		ch: ch,
	}
	return
}

func Release(p Promise, _ Future) {
	pp, ok := p.(promise)
	if ok {
		pool.Put(pp.ch)
	}
}

type value struct {
	val any
	err error
}

type promise struct {
	ch *sch
}

func (p promise) Succeed(v interface{}) {
	p.ch.send(value{
		val: v,
	})
}

func (p promise) Failed(err error) {
	if err == nil {
		err = errors.Warning("fns: empty failed result").WithMeta("fns", "futures")
	}
	p.ch.send(value{
		err: err,
	})
}

type future struct {
	ch *sch
}

func (f future) Get(ctx context.Context) (r Result, err error) {
	select {
	case <-ctx.Done():
		f.ch.destroy()
		err = errors.Timeout("fns: get result value from future timeout").WithMeta("fns", "futures")
		break
	case data, ok := <-f.ch.get():
		pool.Put(f.ch)
		if !ok {
			err = errors.Warning("fns: future was closed").WithMeta("fns", "futures")
			break
		}
		if data.err != nil {
			err = data.err
			break
		}
		r = objects.New(data.val)
		break
	}
	return
}

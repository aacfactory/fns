/*
 * Copyright 2021 Wang Min Xiang
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
 */

package ratelimit

import (
	"github.com/aacfactory/errors"
	"sync/atomic"
	"time"
)

func New(max int64, window time.Duration) *Limiter {
	// todo counter use fastcache (use max tickets to calc max bytes of cache)
	return &Limiter{
		max:    max,
		window: window,
	}
}

type Counter interface {
	Incr(key string, window time.Time) (err error)
	Decr(key string, window time.Time) (err error)
	Get(key string, window time.Time) (n int64)
}

type Limiter struct {
	counter Counter
	max     int64
	window  time.Duration
}

func (limiter *Limiter) getWindow() time.Time {
	return time.Now().Truncate(limiter.window)
}

func (limiter *Limiter) Take(key string) (ok bool, err error) {
	window := limiter.getWindow()
	ok = limiter.counter.Get(key, window) < limiter.max
	if ok {
		err = limiter.counter.Incr(key, window)
		if err != nil {
			err = errors.Warning("fns: limiter take ticket failed").WithCause(err).WithMeta("key", key).WithMeta("window", window.Format(time.RFC3339))
			return
		}
	}
	return
}

func (limiter *Limiter) Repay(key string) (err error) {
	window := limiter.getWindow()
	err = limiter.counter.Decr(key, window)
	if err != nil {
		err = errors.Warning("fns: limiter repay ticket failed").WithCause(err).WithMeta("key", key).WithMeta("window", window.Format(time.RFC3339))
		return
	}
	return
}

func (limiter *Limiter) Close() {

}

type Times struct {
	n int64
}

func (t *Times) Value() int64 {
	return atomic.LoadInt64(&t.n)
}

func (t *Times) Incr() int64 {
	return atomic.AddInt64(&t.n, 1)
}

func (t *Times) Decr() int64 {
	return atomic.AddInt64(&t.n, -1)
}

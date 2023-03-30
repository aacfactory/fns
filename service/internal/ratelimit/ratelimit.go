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
	"sync"
	"sync/atomic"
	"time"
)

func New(max int64, window time.Duration, counter Counter) *Limiter {
	return &Limiter{
		keys:    sync.Map{},
		counter: counter,
		max:     max,
		window:  window,
	}
}

type Counter interface {
	Incr(key string, window time.Time) (err error)
	Decr(key string, window time.Time) (err error)
	Get(key string) (n int64)
}

type Limiter struct {
	keys    sync.Map
	counter Counter
	max     int64
	window  time.Duration
}

func (limiter *Limiter) Take(key string) (ok bool) {
	limiter.counter.Get(key)
	// todo make it as interceptor
	value, _ := limiter.keys.LoadOrStore(key, &Times{})
	times := value.(*Times)
	if times.Value() >= limiter.max {
		return
	}
	times.Incr()
	ok = true
	return
}

func (limiter *Limiter) Repay(key string) {
	value, has := limiter.keys.Load(key)
	if !has {
		return
	}
	times := value.(*Times)
	if times.Decr() <= 0 {
		limiter.keys.Delete(key)
	}
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

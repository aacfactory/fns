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

func New(max int64, window time.Duration) *Limiter {
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
	times  sync.Map
	max    int64
	window time.Duration
}

func (limiter *Limiter) getWindow() int64 {
	if limiter.window == 0 {
		return 0
	}
	return time.Now().Truncate(limiter.window).UnixNano()
}

func (limiter *Limiter) Take(key string) (ok bool, err error) {
	window := limiter.getWindow()
	v, _ := limiter.times.LoadOrStore(key, &Times{
		mu:     sync.Mutex{},
		n:      0,
		window: window,
	})
	times := v.(*Times)
	if times.Window() == window {
		if times.Value() < limiter.max {
			times.Incr()
			ok = true
		}
	} else {
		times.SetWindow(window)
		times.Incr()
		ok = true
	}
	return
}

func (limiter *Limiter) Repay(key string) (err error) {
	window := limiter.getWindow()
	v, loaded := limiter.times.Load(key)
	if !loaded {
		return
	}
	times := v.(*Times)
	if times.Window() == window {
		times.Decr()
	}
	return
}

func (limiter *Limiter) Close() {

}

type Times struct {
	mu     sync.Mutex
	n      int64
	window int64
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

func (t *Times) Window() int64 {
	return atomic.LoadInt64(&t.window)
}

func (t *Times) SetWindow(window int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	atomic.StoreInt64(&t.window, window)
	atomic.StoreInt64(&t.n, 0)
}

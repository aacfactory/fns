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

package caches

import (
	"sync"
	"sync/atomic"
	"time"
)

type Increment struct {
	n        int64
	deadline int64
}

func (incr *Increment) Incr(d int64) int64 {
	return atomic.AddInt64(&incr.n, d)
}

func (incr *Increment) Rest(d int64) {
	atomic.StoreInt64(&incr.n, d)
}

func (incr *Increment) Decr(d int64) int64 {
	return atomic.AddInt64(&incr.n, -1*d)
}

func (incr *Increment) Value() int64 {
	return atomic.LoadInt64(&incr.n)
}

func (incr *Increment) Expired() bool {
	if incr.deadline == 0 {
		return false
	}
	return incr.deadline < time.Now().UnixNano()
}

type Increments struct {
	values sync.Map
}

func (increments *Increments) Incr(k uint64, delta int64) (n int64) {
	v, _ := increments.values.LoadOrStore(k, &Increment{
		n:        0,
		deadline: 0,
	})
	incr := v.(*Increment)
	if incr.Expired() {
		incr.Rest(delta)
		n = delta
	} else {
		n = incr.Incr(delta)
	}
	return
}

func (increments *Increments) Decr(k uint64, delta int64) (n int64) {
	v, _ := increments.values.LoadOrStore(k, &Increment{
		n:        0,
		deadline: 0,
	})
	incr := v.(*Increment)
	if incr.Expired() {
		incr.Rest(delta)
		n = -1 * delta
	} else {
		n = incr.Decr(delta)
	}
	return
}

func (increments *Increments) Value(k uint64) (n int64, has bool) {
	v, exist := increments.values.Load(k)
	if !exist {
		return
	}
	incr := v.(*Increment)
	if incr.Expired() {
		increments.values.Delete(k)
		return
	}
	n = incr.Value()
	has = true
	return
}

func (increments *Increments) Expire(k uint64, ttl time.Duration) {
	v, has := increments.values.Load(k)
	if !has {
		return
	}
	incr := v.(*Increment)
	if ttl <= 0 {
		incr.deadline = 0
	} else {
		incr.deadline = time.Now().Add(ttl).UnixNano()
	}
}

func (increments *Increments) Remove(k uint64) {
	increments.values.Delete(k)
}

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

package lru

import (
	"github.com/aacfactory/errors"
	"sync"
)

type ARCCache[K comparable, V any] struct {
	size int
	p    int
	t1   *LRU[K, V]
	b1   *LRU[K, struct{}]
	t2   *LRU[K, V]
	b2   *LRU[K, struct{}]
	lock sync.RWMutex
}

func NewARC[K comparable, V any](size int) (*ARCCache[K, V], error) {
	if size < 1 {
		return nil, errors.Warning("fns: size is required")
	}
	b1 := New[K, struct{}](size, nil)
	b2 := New[K, struct{}](size, nil)
	t1 := New[K, V](size, nil)
	t2 := New[K, V](size, nil)
	c := &ARCCache[K, V]{
		size: size,
		p:    0,
		t1:   t1,
		b1:   b1,
		t2:   t2,
		b2:   b2,
	}
	return c, nil
}

func (c *ARCCache[K, V]) Get(key K) (value V, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if value, ok = c.t1.Peek(key); ok {
		c.t1.Remove(key)
		c.t2.Add(key, value)
		return value, ok
	}
	if value, ok = c.t2.Get(key); ok {
		return value, ok
	}
	return
}

func (c *ARCCache[K, V]) Add(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.t1.Contains(key) {
		c.t1.Remove(key)
		c.t2.Add(key, value)
		return
	}
	if c.t2.Contains(key) {
		c.t2.Add(key, value)
		return
	}
	if c.b1.Contains(key) {
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b2Len > b1Len {
			delta = b2Len / b1Len
		}
		if c.p+delta >= c.size {
			c.p = c.size
		} else {
			c.p += delta
		}
		if c.t1.Len()+c.t2.Len() >= c.size {
			c.replace(false)
		}
		c.b1.Remove(key)
		c.t2.Add(key, value)
		return
	}
	if c.b2.Contains(key) {
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b1Len > b2Len {
			delta = b1Len / b2Len
		}
		if delta >= c.p {
			c.p = 0
		} else {
			c.p -= delta
		}
		if c.t1.Len()+c.t2.Len() >= c.size {
			c.replace(true)
		}
		c.b2.Remove(key)
		c.t2.Add(key, value)
		return
	}
	if c.t1.Len()+c.t2.Len() >= c.size {
		c.replace(false)
	}
	if c.b1.Len() > c.size-c.p {
		c.b1.RemoveOldest()
	}
	if c.b2.Len() > c.p {
		c.b2.RemoveOldest()
	}
	c.t1.Add(key, value)
}

func (c *ARCCache[K, V]) replace(b2ContainsKey bool) {
	t1Len := c.t1.Len()
	if t1Len > 0 && (t1Len > c.p || (t1Len == c.p && b2ContainsKey)) {
		k, _, ok := c.t1.RemoveOldest()
		if ok {
			c.b1.Add(k, struct{}{})
		}
	} else {
		k, _, ok := c.t2.RemoveOldest()
		if ok {
			c.b2.Add(k, struct{}{})
		}
	}
}

func (c *ARCCache[K, V]) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.t1.Len() + c.t2.Len()
}

func (c *ARCCache[K, V]) Keys() []K {
	c.lock.RLock()
	defer c.lock.RUnlock()
	k1 := c.t1.Keys()
	k2 := c.t2.Keys()
	return append(k1, k2...)
}

func (c *ARCCache[K, V]) Values() []V {
	c.lock.RLock()
	defer c.lock.RUnlock()
	v1 := c.t1.Values()
	v2 := c.t2.Values()
	return append(v1, v2...)
}

func (c *ARCCache[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.t1.Remove(key) {
		return
	}
	if c.t2.Remove(key) {
		return
	}
	if c.b1.Remove(key) {
		return
	}
	if c.b2.Remove(key) {
		return
	}
}

func (c *ARCCache[K, V]) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.t1.Purge()
	c.t2.Purge()
	c.b1.Purge()
	c.b2.Purge()
}

func (c *ARCCache[K, V]) Contains(key K) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.t1.Contains(key) || c.t2.Contains(key)
}

func (c *ARCCache[K, V]) Peek(key K) (value V, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if val, ok := c.t1.Peek(key); ok {
		return val, ok
	}
	return c.t2.Peek(key)
}

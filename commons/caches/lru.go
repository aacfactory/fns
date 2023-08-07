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
	"container/list"
	"sync"
	"time"
)

type ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

type LRU[K ordered, V any] struct {
	mu        sync.RWMutex
	size      int64
	evictList *list.List
	items     map[K]*list.Element
}

type lruItem[K ordered, V any] struct {
	key      K
	value    V
	expireAT time.Time
}

func NewLRU[K ordered, V any](size uint32) (lru *LRU[K, V]) {
	lru = &LRU[K, V]{
		mu:        sync.RWMutex{},
		size:      int64(size),
		evictList: list.New(),
		items:     make(map[K]*list.Element),
	}
	return
}

func (c *LRU[K, V]) Purge() {
	c.mu.Lock()
	for k := range c.items {
		delete(c.items, k)
	}
	c.evictList.Init()
	c.mu.Unlock()
}

func (c *LRU[K, V]) Add(key K, value V, ttl time.Duration) (evicted bool) {
	c.mu.Lock()
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*lruItem[K, V]).value = value
		c.mu.Unlock()
		return
	}
	expireAT := time.Time{}
	if ttl > 0 {
		expireAT = time.Now().Add(ttl)
	}
	ent := &lruItem[K, V]{key, value, expireAT}
	c.items[key] = c.evictList.PushFront(ent)
	c.mu.Unlock()
	evicted = int64(c.evictList.Len()) > c.size
	if evicted {
		c.removeOldest()
	}
	return
}

func (c *LRU[K, V]) Get(key K) (value V, ok bool) {
	c.mu.RLock()
	if ent, has := c.items[key]; has {
		c.evictList.MoveToFront(ent)
		if ent.Value.(*lruItem[K, V]) == nil {
			return
		}
		item := ent.Value.(*lruItem[K, V])
		if item.expireAT.IsZero() {
			value = item.value
			ok = true
			c.mu.RUnlock()
			return
		}
		ok = item.expireAT.After(time.Now())
		if ok {
			value = item.value
			c.mu.RUnlock()
		} else {
			c.mu.RUnlock()
			c.removeElement(ent)
		}
		return
	}
	c.mu.RUnlock()
	return
}

func (c *LRU[K, V]) Remove(key K) (present bool) {
	c.mu.RLock()
	if ent, ok := c.items[key]; ok {
		c.mu.RUnlock()
		c.removeElement(ent)
		present = true
		return
	}
	c.mu.RUnlock()
	return
}

func (c *LRU[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]K, len(c.items))
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys[i] = ent.Value.(*lruItem[K, V]).key
		i++
	}
	return keys
}

func (c *LRU[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.evictList.Len()
}

func (c *LRU[K, V]) Resize(size int64) (evicted int64) {
	evicted = int64(c.Len()) - size
	if evicted < 0 {
		evicted = 0
	}
	for i := int64(0); i < evicted; i++ {
		c.removeOldest()
	}
	c.size = size
	return
}

func (c *LRU[K, V]) removeOldest() {
	c.mu.RLock()
	ent := c.evictList.Back()
	c.mu.RUnlock()
	if ent != nil {
		c.removeElement(ent)
	}
}

func (c *LRU[K, V]) removeElement(e *list.Element) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictList.Remove(e)
	kv := e.Value.(*lruItem[K, V])
	delete(c.items, kv.key)
}

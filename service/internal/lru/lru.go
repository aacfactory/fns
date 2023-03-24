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

package lru

import (
	"container/list"
	"time"
)

type LRU[K any, V any] struct {
	size      int64
	evictList *list.List
	items     map[interface{}]*list.Element
}

type lruItem[K any, V any] struct {
	key      K
	value    V
	expireAT time.Time
}

func New[K any, V any](size uint32) (lru *LRU[K, V]) {
	lru = &LRU[K, V]{
		size:      int64(size),
		evictList: list.New(),
		items:     make(map[interface{}]*list.Element),
	}
	return
}

func (c *LRU[K, V]) Purge() {
	for k := range c.items {
		delete(c.items, k)
	}
	c.evictList.Init()
}

func (c *LRU[K, V]) Add(key K, value V, ttl time.Duration) (evicted bool) {
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.(*lruItem[K, V]).value = value
		return
	}
	expireAT := time.Time{}
	if ttl > 0 {
		expireAT = time.Now().Add(ttl)
	}
	ent := &lruItem[K, V]{key, value, expireAT}
	c.items[key] = c.evictList.PushFront(ent)

	evicted = int64(c.evictList.Len()) > c.size
	if evicted {
		c.removeOldest()
	}
	return
}

func (c *LRU[K, V]) Get(key K) (value V, ok bool) {
	if ent, has := c.items[key]; has {
		c.evictList.MoveToFront(ent)
		if ent.Value.(*lruItem[K, V]) == nil {
			return
		}
		item := ent.Value.(*lruItem[K, V])
		if item.expireAT.IsZero() {
			value = item.value
			ok = true
			return
		}
		ok = item.expireAT.After(time.Now())
		if ok {
			value = item.value
		} else {
			c.removeElement(ent)
		}
		return
	}
	return
}

func (c *LRU[K, V]) Remove(key K) (present bool) {
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		present = true
		return
	}
	return
}

func (c *LRU[K, V]) Keys() []K {
	keys := make([]K, len(c.items))
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys[i] = ent.Value.(*lruItem[K, V]).key
		i++
	}
	return keys
}

func (c *LRU[K, V]) Len() int {
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
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

func (c *LRU[K, V]) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*lruItem[K, V])
	delete(c.items, kv.key)
}

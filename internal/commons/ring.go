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

package commons

import (
	"github.com/valyala/bytebufferpool"
	"sync"
)

type Keyable interface {
	Key() (key string)
}

type RingEntry struct {
	next, prev *RingEntry
	value      Keyable
}

func NewRing(values ...Keyable) (r *Ring) {
	r = &Ring{
		mutex: sync.RWMutex{},
		head:  nil,
		size:  0,
	}
	if values != nil && len(values) > 0 {
		for _, value := range values {
			r.Append(value)
		}
	}
	return
}

type Ring struct {
	mutex sync.RWMutex
	head  *RingEntry
	size  int
}

func (r *Ring) Append(v Keyable) {
	if v == nil {
		return
	}
	r.mutex.Lock()
	entry := &RingEntry{
		value: v,
	}
	if r.head == nil {
		entry.next = entry
		entry.prev = entry
		r.head = entry
	} else {
		prev := r.head.prev
		prev.next = entry
		entry.prev = prev
		entry.next = r.head
		r.head.prev = entry
	}
	r.size++
	r.mutex.Unlock()
}

func (r *Ring) Remove(v Keyable) {
	if v == nil {
		return
	}
	r.mutex.Lock()
	if r.head == nil {
		r.mutex.Unlock()
		return
	}
	for i := 0; i < r.size; i++ {
		entry := r.next()
		if entry.value.Key() == v.Key() {
			if entry.prev.value.Key() == v.Key() && entry.next.value.Key() == v.Key() {
				r.head = nil
				break
			}
			prev := entry.prev
			next := entry.next
			prev.next = next
			break
		}
	}
	r.size--
	r.mutex.Unlock()
}

func (r *Ring) Next() (value Keyable) {
	r.mutex.RLock()
	if r.size == 0 {
		r.mutex.RUnlock()
		return
	}
	value = r.next().value
	r.mutex.RUnlock()
	return
}

func (r *Ring) Get(key string) (value Keyable) {
	r.mutex.RLock()
	if r.size == 0 {
		r.mutex.RUnlock()
		return
	}
	for i := 0; i < r.size; i++ {
		n := r.next().value
		if n.Key() == key {
			value = n
			break
		}
	}
	r.mutex.RUnlock()
	return
}

func (r *Ring) Size() (size int) {
	r.mutex.RLock()
	size = r.size
	r.mutex.RUnlock()
	return
}

func (r *Ring) String() (value string) {
	r.mutex.RLock()
	p := bytebufferpool.Get()
	_ = p.WriteByte('[')
	for i := 0; i < r.size; i++ {
		entry := r.next()
		if i == 0 {
			_, _ = p.WriteString(entry.value.Key())
		} else {
			_, _ = p.WriteString(", ")
			_, _ = p.WriteString(entry.value.Key())
		}
	}
	_ = p.WriteByte(']')
	value = p.String()
	bytebufferpool.Put(p)
	r.mutex.RUnlock()
	return
}

func (r *Ring) next() (entry *RingEntry) {
	entry = r.head
	r.head = r.head.next
	return
}

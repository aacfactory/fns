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

package bmap

import (
	"cmp"
	"sort"
)

type Entry[Key cmp.Ordered, Value any] struct {
	Key    Key
	Values []Value
}

type Entries[Key cmp.Ordered, Value any] []Entry[Key, Value]

func (e Entries[Key, Value]) Len() int {
	return len(e)
}

func (e Entries[Key, Value]) Less(i, j int) bool {
	return cmp.Less(e[i].Key, e[j].Key)
}

func (e Entries[Key, Value]) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func New[Key cmp.Ordered, Value any]() BMap[Key, Value] {
	return make(BMap[Key, Value], 0, 1)
}

type BMap[Key cmp.Ordered, Value any] Entries[Key, Value]

func (b *BMap[Key, Value]) Set(key Key, val Value) {
	s := *b
	for i, entry := range s {
		if cmp.Compare[Key](entry.Key, key) == 0 {
			s[i] = Entry[Key, Value]{
				Key:    key,
				Values: []Value{val},
			}
			*b = s
			return
		}
	}
	s = append(s, Entry[Key, Value]{
		Key:    key,
		Values: []Value{val},
	})
	sort.Sort(Entries[Key, Value](s))
	*b = s
}

func (b *BMap[Key, Value]) Add(key Key, val Value) {
	s := *b
	for _, entry := range s {
		if cmp.Compare[Key](entry.Key, key) == 0 {
			entry.Values = append(entry.Values, val)
			*b = s
			return
		}
	}
	s = append(s, Entry[Key, Value]{
		Key:    key,
		Values: []Value{val},
	})
	sort.Sort(Entries[Key, Value](s))
	*b = s
}

func (b *BMap[Key, Value]) Get(key Key) (v Value, found bool) {
	s := *b
	bLen := b.Len()
	if bLen < 65 {
		for _, entry := range s {
			if cmp.Compare[Key](entry.Key, key) == 0 {
				v = entry.Values[0]
				found = true
				return
			}
		}
		return
	}
	n := bLen
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if cmp.Less(s[h].Key, key) {
			i = h + 1
		} else {
			j = h
		}
	}
	found = i < n
	if found {
		v = s[i].Values[0]
	}
	return
}

func (b *BMap[Key, Value]) Values(key Key) ([]Value, bool) {
	s := *b
	for _, entry := range s {
		if cmp.Compare[Key](entry.Key, key) == 0 {
			return entry.Values, true
		}
	}
	return nil, false
}

func (b *BMap[Key, Value]) Remove(key Key) {
	s := *b
	n := -1
	for i, entry := range s {
		if cmp.Compare[Key](entry.Key, key) == 0 {
			n = i
			break
		}
	}
	if n > -1 {
		s = append(s[:n], s[n+1:]...)
		*b = s
	}
}

func (b *BMap[Key, Value]) Foreach(fn func(key Key, values []Value)) {
	s := *b
	for _, entry := range s {
		fn(entry.Key, entry.Values)
	}
}

func (b *BMap[Key, Value]) Len() int {
	return len(*b)
}

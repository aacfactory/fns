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

import "time"

type Entry[K comparable, V any] struct {
	next         *Entry[K, V]
	prev         *Entry[K, V]
	list         *List[K, V]
	Key          K
	Value        V
	ExpiresAt    time.Time
	ExpireBucket uint8
}

func (e *Entry[K, V]) PrevEntry() *Entry[K, V] {
	if p := e.prev; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

type List[K comparable, V any] struct {
	root Entry[K, V]
	len  int
}

func (l *List[K, V]) Init() *List[K, V] {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

func NewList[K comparable, V any]() *List[K, V] { return new(List[K, V]).Init() }

func (l *List[K, V]) Length() int { return l.len }

func (l *List[K, V]) Back() *Entry[K, V] {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

func (l *List[K, V]) lazyInit() {
	if l.root.next == nil {
		l.Init()
	}
}

func (l *List[K, V]) insert(e, at *Entry[K, V]) *Entry[K, V] {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.list = l
	l.len++
	return e
}

func (l *List[K, V]) insertValue(k K, v V, expiresAt time.Time, at *Entry[K, V]) *Entry[K, V] {
	return l.insert(&Entry[K, V]{Value: v, Key: k, ExpiresAt: expiresAt}, at)
}

func (l *List[K, V]) Remove(e *Entry[K, V]) V {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--

	return e.Value
}

func (l *List[K, V]) move(e, at *Entry[K, V]) {
	if e == at {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev

	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
}

func (l *List[K, V]) PushFront(k K, v V) *Entry[K, V] {
	l.lazyInit()
	return l.insertValue(k, v, time.Time{}, &l.root)
}

func (l *List[K, V]) PushFrontExpirable(k K, v V, expiresAt time.Time) *Entry[K, V] {
	l.lazyInit()
	return l.insertValue(k, v, expiresAt, &l.root)
}

func (l *List[K, V]) MoveToFront(e *Entry[K, V]) {
	if e.list != l || l.root.next == e {
		return
	}
	l.move(e, &l.root)
}

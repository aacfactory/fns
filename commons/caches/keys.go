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

import "sync"

const minFill = 30

func NewKeys() *Keys {
	return &Keys{
		mu:       sync.RWMutex{},
		min:      0,
		max:      0,
		items:    make([]uint64, 0, 1),
		mask:     0,
		shrinkAt: 0,
		len:      0,
	}
}

type Keys struct {
	mu       sync.RWMutex
	min, max uint64
	items    []uint64
	mask     uint64
	shrinkAt int
	len      int
}

func (m *Keys) grow() {
	items := make([]uint64, len(m.items)*2)
	mask := uint64(len(items) - 1)
	for idx := m.min; idx <= m.max; idx++ {
		if m.items[idx&m.mask] == idx && m.items[idx&m.mask] != 0 {
			items[idx&mask] = m.items[idx&m.mask]
		}
	}
	m.items = items
	m.mask = uint64(len(m.items) - 1)
	m.shrinkAt = len(m.items) * minFill / 100
}

func (m *Keys) shrink() {
	sz := 1
	for sz < m.shrinkAt {
		sz *= 2
	}
	items := make([]uint64, sz)
	mask := uint64(len(items) - 1)
	for idx := m.min; idx <= m.max; idx++ {
		if m.items[idx&m.mask] == idx && m.items[idx&m.mask] != 0 {
			items[idx&mask] = m.items[idx&m.mask]
		}
	}
	m.items = items
	m.mask = uint64(len(m.items) - 1)
	m.shrinkAt = len(m.items) * minFill / 100
}

func (m *Keys) Set(key uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.items) == 0 {
		m.items = make([]uint64, 1)
		m.mask = uint64(len(m.items) - 1)
		m.shrinkAt = len(m.items) * minFill / 100
		m.min, m.max = key, key
		m.items[0] = key
		m.len = 1
		return
	}
	if m.items[key&m.mask] != 0 {
		if m.items[key&m.mask] == key {
			return
		}
		for m.items[key&m.mask] != 0 {
			m.grow()
		}
	}
	if key < m.min {
		m.min = key
	} else if key > m.max {
		m.max = key
	}
	m.items[key&m.mask] = key
	m.len++
	return
}

func (m *Keys) Exist(key uint64) (has bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.items) == 0 {
		return
	}
	if m.items[key&m.mask] == key {
		has = true
		return
	}
	return
}

func (m *Keys) Remove(key uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.items) == 0 || m.items[key&m.mask] != key || m.items[key&m.mask] == 0 {
		return
	}
	m.items[key&m.mask] = 0
	m.len--
	if m.min == key {
		if m.max == key {
			m.min, m.max = 0, 0
			m.items = nil
			return
		}
		for {
			m.min++
			if m.items[(m.min)&m.mask] == 0 {
				continue
			}
			break
		}
	} else if m.max == key {
		for m.max > m.min {
			m.max--
			if m.items[(m.max)&m.mask] == 0 {
				continue
			}
			break
		}
	}
	if (m.max - m.min + 1) <= uint64(m.shrinkAt) {
		m.shrink()
	}
	return
}

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

package smap

const minFill = 30

type mapItem struct {
	index uint64
	item  interface{}
}

func New() *Map {
	return &Map{
		min:      0,
		max:      0,
		items:    make([]mapItem, 0, 1),
		mask:     0,
		shrinkAt: 0,
		len:      0,
	}
}

type Map struct {
	min, max uint64
	items    []mapItem
	mask     uint64
	shrinkAt int
	len      int
}

func (m *Map) grow() {
	items := make([]mapItem, len(m.items)*2)
	mask := uint64(len(items) - 1)
	for idx := m.min; idx <= m.max; idx++ {
		if m.items[idx&m.mask].index == idx &&
			m.items[idx&m.mask].item != nil {
			items[idx&mask] = m.items[idx&m.mask]
		}
	}
	m.items = items
	m.mask = uint64(len(m.items) - 1)
	m.shrinkAt = len(m.items) * minFill / 100
}

func (m *Map) shrink() {
	sz := 1
	for sz < m.shrinkAt {
		sz *= 2
	}
	items := make([]mapItem, sz)
	mask := uint64(len(items) - 1)
	for idx := m.min; idx <= m.max; idx++ {
		if m.items[idx&m.mask].index == idx &&
			m.items[idx&m.mask].item != nil {
			items[idx&mask] = m.items[idx&m.mask]
		}
	}
	m.items = items
	m.mask = uint64(len(m.items) - 1)
	m.shrinkAt = len(m.items) * minFill / 100
}

func (m *Map) Set(index uint64, item interface{}) {
	if len(m.items) == 0 {
		m.items = make([]mapItem, 1)
		m.mask = uint64(len(m.items) - 1)
		m.shrinkAt = len(m.items) * minFill / 100
		m.min, m.max = index, index
		m.items[0].index = index
		m.items[0].item = item
		m.len = 1
		return
	}
	if m.items[index&m.mask].item != nil {
		if m.items[index&m.mask].index == index {
			m.items[index&m.mask].item = item
			return
		}
		for m.items[index&m.mask].item != nil {
			m.grow()
		}
	}
	if index < m.min {
		m.min = index
	} else if index > m.max {
		m.max = index
	}
	m.items[index&m.mask].index = index
	m.items[index&m.mask].item = item
	m.len++
	return
}

func (m *Map) Get(index uint64) (value interface{}, has bool) {
	if len(m.items) == 0 {
		return
	}
	if m.items[index&m.mask].index == index {
		value = m.items[index&m.mask].item
		has = true
		return
	}
	return
}

func (m *Map) Delete(index uint64) {
	if len(m.items) == 0 || m.items[index&m.mask].index != index ||
		m.items[index&m.mask].item == nil {
		return
	}
	m.items[index&m.mask].index = 0
	m.items[index&m.mask].item = nil
	m.len--
	if m.min == index {
		if m.max == index {
			m.min, m.max = 0, 0
			m.items = nil
			return
		}
		for {
			m.min++
			if m.items[(m.min)&m.mask].item == nil {
				continue
			}
			break
		}
	} else if m.max == index {
		for m.max > m.min {
			m.max--
			if m.items[(m.max)&m.mask].item == nil {
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

func (m *Map) Len() int {
	return m.len
}

func (m *Map) Min() uint64 {
	return m.min
}

func (m *Map) Max() uint64 {
	return m.max
}

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

package services

import (
	"sort"
	"strings"
	"unsafe"
)

type FnInfo struct {
	Name     string `json:"name"`
	Readonly bool   `json:"readonly"`
	Internal bool   `json:"internal"`
}

type FnInfos []FnInfo

func (f FnInfos) Len() int {
	return len(f)
}

func (f FnInfos) Less(i, j int) bool {
	return strings.Compare(f[i].Name, f[j].Name) < 0
}

func (f FnInfos) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f FnInfos) Find(name []byte) (info FnInfo, found bool) {
	ns := unsafe.String(unsafe.SliceData(name), len(name))
	n := f.Len()
	if n < 65 {
		for _, fn := range f {
			if fn.Name == ns {
				info = fn
				found = true
				break
			}
		}
		return
	}
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if strings.Compare(f[h].Name, ns) < 0 {
			i = h + 1
		} else {
			j = h
		}
	}
	found = i < n && f[i].Name == ns
	if found {
		info = f[i]
	}
	return
}

type Fn interface {
	Name() string
	Internal() bool
	Readonly() bool
	Handle(r Request) (v interface{}, err error)
}

type Fns []Fn

func (f Fns) Len() int {
	return len(f)
}

func (f Fns) Less(i, j int) bool {
	return strings.Compare(f[i].Name(), f[j].Name()) < 0
}

func (f Fns) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f Fns) Add(v Fn) Fns {
	ff := append(f, v)
	sort.Sort(ff)
	return ff
}

func (f Fns) Find(name []byte) (v Fn, found bool) {
	ns := unsafe.String(unsafe.SliceData(name), len(name))
	n := f.Len()
	if n < 65 {
		for _, fn := range f {
			if fn.Name() == ns {
				v = fn
				found = true
				break
			}
		}
		return
	}
	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)
		if strings.Compare(f[h].Name(), ns) < 0 {
			i = h + 1
		} else {
			j = h
		}
	}
	found = i < n && f[i].Name() == ns
	if found {
		v = f[i]
	}
	return
}

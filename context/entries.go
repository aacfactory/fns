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

package context

import (
	"bytes"
)

type Entry struct {
	key []byte
	val any
}

type Entries []Entry

func (entries *Entries) Get(key []byte) (val any) {
	s := *entries
	for _, entry := range s {
		if bytes.Equal(key, entry.key) {
			val = entry.val
			return
		}
	}
	return
}

func (entries *Entries) Set(key []byte, val any) {
	s := *entries
	for _, entry := range s {
		if bytes.Equal(key, entry.key) {
			entry.val = val
			*entries = s
			return
		}
	}
	s = append(s, Entry{
		key: key,
		val: val,
	})
	*entries = s
	return
}

func (entries *Entries) Remove(key []byte) {
	s := *entries
	n := -1
	for i, entry := range s {
		if bytes.Equal(key, entry.key) {
			n = i
			break
		}
	}
	if n > -1 {
		s = append(s[:n], s[n+1:]...)
		*entries = s
	}
	return
}

func (entries *Entries) Foreach(fn func(key []byte, value any)) {
	s := *entries
	for _, entry := range s {
		fn(entry.key, entry.val)
	}
}

func (entries *Entries) Len() int {
	return len(*entries)
}

func (entries *Entries) Reset() {
	s := *entries
	s = s[:0]
	*entries = s
}

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

package fns

type MultiMap map[string][]string

func (m MultiMap) Add(key string, value string) {
	m[key] = append(m[key], value)
}

func (m MultiMap) Put(key string, value []string) {
	m[key] = value
}

func (m MultiMap) Get(key string) (string, bool) {
	if m == nil {
		return "", false
	}
	if v, has := m[key]; has && v != nil {
		return v[0], true
	}
	return "", false
}

func (m MultiMap) Values(key string) ([]string, bool) {
	v, has := m[key]
	return v, has
}

func (m MultiMap) Remove(key string) {
	delete(m, key)
}

func (m MultiMap) Keys() []string {
	if m.Empty() {
		return nil
	}
	keys := make([]string, 0, 1)
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func (m MultiMap) Empty() bool {
	return m == nil || len(m) == 0
}

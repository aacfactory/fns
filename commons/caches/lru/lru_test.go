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

package lru_test

import (
	"github.com/aacfactory/fns/commons/caches/lru"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cache := lru.NewWithExpire[int, int](5, 10*time.Millisecond, func(key int, value int) {
		t.Log("evict:", key, value)
	})
	cache.Add(1, 1)
	cache.Add(2, 2)
	t.Log("len:", cache.Len())
	t.Log(cache.Get(1))
	t.Log(cache.Get(2))
	cache.Remove(2)
	t.Log(cache.Get(2))
	time.Sleep(1 * time.Second)
	t.Log(cache.Get(1))
}

func TestNewWithNoExpire(t *testing.T) {
	cache := lru.New[int, int](5, func(key int, value int) {
		t.Log("evict:", key, value)
	})
	cache.Add(1, 1)
	cache.Add(2, 2)
	t.Log("len:", cache.Len())
	t.Log(cache.Get(1))
	t.Log(cache.Get(2))
	cache.Remove(2)
	t.Log(cache.Get(2))
	time.Sleep(1 * time.Second)
	t.Log(cache.Get(1))
}

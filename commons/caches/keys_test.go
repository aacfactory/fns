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

package caches_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/caches"
	"sync/atomic"
	"testing"
)

func TestKeys(t *testing.T) {
	keys := caches.Keys{}
	for i := 0; i < 10; i++ {
		keys.Set(uint64(i + 1))
	}
	keys.Remove(5)
	keys.Remove(33)
	for i := 0; i < 11; i++ {
		fmt.Println((i + 1), keys.Exist(uint64(i+1)))
	}
}

func BenchmarkKeys_Set(b *testing.B) {
	keys := caches.Keys{}
	n := uint64(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			nn := atomic.LoadUint64(&n)
			if nn%2 == 0 {
				_ = keys.Exist(nn)
			} else {
				keys.Set(nn)
			}
			atomic.AddUint64(&n, 1)
		}
	})
}

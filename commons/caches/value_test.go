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

package caches_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/caches"
	"testing"
	"time"
)

func TestMakeKVS(t *testing.T) {
	big := [2 << 16]byte{}
	kvs := caches.MakeKVS([]byte("s"), big[:], 10*time.Second, caches.MemHash{})
	fmt.Println(len(kvs))
	fmt.Println(kvs.Deadline())
	fmt.Println(len(kvs.Value()), len(big))
	for _, kv := range kvs {
		kLen := len(kv.Key())
		vLen := len(kv.Value())
		ok := (4 + kLen + vLen) < (1 << 16)
		fmt.Println("key:", kLen, "val:", vLen, ok)
	}
}

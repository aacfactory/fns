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
	"github.com/aacfactory/fns/commons/uid"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cache := caches.New(32 * 1024 * 1024)
	keyA := []byte("a")
	setKeyAErr := cache.Set(keyA, keyA)
	if setKeyAErr != nil {
		t.Error(setKeyAErr)
		return
	}
	valA, hasA := cache.Get(keyA)
	fmt.Println(hasA, string(valA))
	// big
	keyB := []byte("b")
	big := [2 << 16]byte{}
	copy(big[0:1], []byte{'b'})
	copy(big[len(big)-1:], []byte{'b'})
	setKeyBErr := cache.Set(keyB, big[:])
	if setKeyBErr != nil {
		t.Error(setKeyBErr)
		return
	}
	valB, hasB := cache.Get(keyB)
	fmt.Println(hasB, string(valB))

	// ttl
	keyC := []byte("c")
	setKeyCErr := cache.SetWithTTL(keyC, keyC, 1*time.Second)
	if setKeyCErr != nil {
		t.Error(setKeyCErr)
		return
	}
	valC, hasC := cache.Get(keyC)
	fmt.Println(hasC, string(valC))
	time.Sleep(1 * time.Second)
	valC, hasC = cache.Get(keyC)
	fmt.Println(hasC, string(valC))
	cache.Remove(keyB)
}

func TestIncr(t *testing.T) {
	cache := caches.New(32 * 1024 * 1024)
	key := []byte("a")
	for i := 0; i < 10; i++ {
		fmt.Println(cache.Incr(key, 1))
	}
	fmt.Println(cache.Expire(key, 10*time.Second))
	cache.Remove(key)
	for i := 0; i < 10; i++ {
		fmt.Println(cache.Decr(key, 1))
	}
}

func TestCache_Set(t *testing.T) {
	cache := caches.New(32 * 1024 * 1024)
	key := []byte("a")
	val := []byte(uid.UID())
	fmt.Println(cache.Set(key, val))
}

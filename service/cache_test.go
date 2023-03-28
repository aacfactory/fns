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

package service_test

import (
	"fmt"
	"github.com/aacfactory/fns/service"
	"github.com/dgraph-io/ristretto"
	"net/http"
	"testing"
)

func TestCacheControl_GetMaxAge(t *testing.T) {
	cache := &service.CacheControl{}
	header := http.Header{}
	header.Set("Cache-Control", "max-age=10")
	age, has := cache.MaxAge(header)
	fmt.Println(age, has)
}

func TestCache(t *testing.T) {
	tags, createCacheErr := ristretto.NewCache(&ristretto.Config{
		NumCounters: 10000,
		MaxCost:     int64(64),
		BufferItems: 64,
		Metrics:     false,
		OnEvict: func(item *ristretto.Item) {
			fmt.Println("evict:", item.Key)
		},
		OnReject: func(item *ristretto.Item) {
			fmt.Println("reject:", item.Key)
		},
		OnExit:             nil,
		KeyToHash:          nil,
		Cost:               nil,
		IgnoreInternalCost: false,
	})
	if createCacheErr != nil {
		t.Errorf("%+v", createCacheErr)
		return
	}

	fmt.Println(tags.Set(uint64(1), "12345", 5))
	tags.Wait()
	fmt.Println(tags.Get(uint64(1)))
	fmt.Println(tags.Set(uint64(2), "12345", 5))
	fmt.Println(tags.Get(uint64(1)))
	fmt.Println(tags.Set(uint64(3), "12345", 5))

	fmt.Println(tags.Get(uint64(1)))
	fmt.Println("wait")
	tags.Close()

}

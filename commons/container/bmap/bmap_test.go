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

package bmap_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/container/bmap"
	"testing"
)

func TestNewBMap(t *testing.T) {
	bm := bmap.New[string, []byte]()
	bm.Set("a", []byte("a"))
	bm.Set("c", []byte("c"))
	bm.Add("b", []byte("b1"))
	bm.Add("b", []byte("b2"))
	bm.Foreach(func(key string, values [][]byte) {
		fmt.Println(key)
	})
	v, has := bm.Get("c")
	fmt.Println(string(v), has)
	bm.Remove("c")
	v, has = bm.Get("c")
	fmt.Println(string(v), has)
	vv, has := bm.Values("b")
	fmt.Println(vv, has)
}

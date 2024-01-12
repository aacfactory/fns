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

package futures_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns/commons/futures"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	p, f := futures.New()
	go func(wg *sync.WaitGroup, p futures.Promise) {
		p.Succeed(1)
		wg.Done()
	}(wg, p)

	wg.Wait()
	r, err := f.Await(context.TODO())
	if err != nil {
		fmt.Println(err)
		return
	}
	v := 0
	err = r.Unmarshal(&v)
	fmt.Println(v, err)
}

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p, f := futures.New()
		p.Succeed(1)
		_, _ = f.Await(context.TODO())
	}
}

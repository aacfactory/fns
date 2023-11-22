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

package context_test

import (
	"bytes"
	sc "context"
	"fmt"
	"github.com/aacfactory/fns/context"
	"testing"
)

func BenchmarkE(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bytes.Equal([]byte{'1'}, []byte{'2'})
	}
}

func BenchmarkEs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bytes.Equal([]byte{'1'}, []byte{'2'})
	}
}

func TestAcquire(t *testing.T) {
	ctx := context.Acquire(sc.Background())
	ctx = context.WithValue(ctx, []byte{'1'}, 1)
	set(ctx)
	ctx.UserValues(func(key []byte, val any) {
		fmt.Println(string(key), val)
	})
	context.Release(ctx)
}

func set(ctx context.Context) {
	context.WithValue(ctx, []byte{'2'}, 2)
}

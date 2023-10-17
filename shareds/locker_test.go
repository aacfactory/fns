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

package shareds_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns/shareds"
	"sync"
	"testing"
	"time"
)

type N struct {
	n int
}

func TestLocalLockers(t *testing.T) {
	lockers := shareds.LocalLockers()
	ctx := context.TODO()
	wg := sync.WaitGroup{}
	n := &N{
		n: 0,
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(ctx context.Context, lockers shareds.Lockers, group *sync.WaitGroup, x *N) {
			defer group.Done()
			locker, getErr := lockers.Acquire(ctx, []byte("locker"), 2*time.Second)
			if getErr != nil {
				t.Errorf("%+v", getErr)
				return
			}
			lockErr := locker.Lock(ctx)
			if lockErr != nil {
				t.Errorf("%+v", lockErr)
				return
			}
			x.n++
			fmt.Println(x.n)
			_ = locker.Unlock(ctx)
		}(ctx, lockers, &wg, n)
	}
	wg.Wait()
}

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

package spinlock

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type Locker struct {
	_    sync.Mutex
	lock uintptr
}

func (l *Locker) Lock() {
	backoff := 1
	for !l.TryLock() {
		for i := 0; i < backoff; i++ {
			runtime.Gosched()
		}
		if backoff < 128 {
			backoff *= 2
		}
	}
}

func (l *Locker) Unlock() {
	atomic.StoreUintptr(&l.lock, 0)
}

func (l *Locker) TryLock() bool {
	return atomic.CompareAndSwapUintptr(&l.lock, 0, 1)
}
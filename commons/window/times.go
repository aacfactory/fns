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

package window

import (
	"sync/atomic"
	"time"
)

func NewTimes(win time.Duration) *Times {
	return &Times{
		n:        0,
		window:   win,
		deadline: time.Now().Truncate(win),
	}
}

type Times struct {
	n        int64
	window   time.Duration
	deadline time.Time
}

func (times *Times) Incr() int64 {
	if times.deadline.Before(time.Now()) {
		times.deadline = time.Now().Truncate(times.window)
		atomic.StoreInt64(&times.n, 1)
		return 1
	}
	return atomic.AddInt64(&times.n, 1)
}

func (times *Times) Decr() int64 {
	if times.deadline.Before(time.Now()) {
		times.deadline = time.Now().Truncate(times.window)
		atomic.StoreInt64(&times.n, 0)
		return 0
	}
	return atomic.AddInt64(&times.n, -1)
}

func (times *Times) Value() int64 {
	return atomic.LoadInt64(&times.n)
}

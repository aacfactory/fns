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

package commons

import "sync/atomic"

func NewSafeFlag(on bool) *SafeFlag {
	return &SafeFlag{
		value: func(on bool) int64 {
			if on {
				return int64(1)
			}
			return int64(0)
		}(on),
	}
}

type SafeFlag struct {
	value int64
}

func (f *SafeFlag) On() {
	atomic.StoreInt64(&f.value, 1)
}

func (f *SafeFlag) Off() {
	atomic.StoreInt64(&f.value, 0)
}

func (f *SafeFlag) IsOn() (ok bool) {
	ok = atomic.LoadInt64(&f.value) == 1
	return
}

func (f *SafeFlag) IsOff() (ok bool) {
	ok = atomic.LoadInt64(&f.value) == 0
	return
}

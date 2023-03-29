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

package flags

import "sync/atomic"

func New(on bool) *Flag {
	return &Flag{
		value: func(on bool) int64 {
			if on {
				return int64(1)
			}
			return int64(0)
		}(on),
	}
}

type Flag struct {
	value int64
}

func (f *Flag) HalfOn() {
	atomic.StoreInt64(&f.value, 1)
}

func (f *Flag) On() {
	atomic.StoreInt64(&f.value, 2)
}

func (f *Flag) Off() {
	atomic.StoreInt64(&f.value, 0)
}

func (f *Flag) IsOn() (ok bool) {
	ok = atomic.LoadInt64(&f.value) >= 1
	return
}

func (f *Flag) IsHalfOn() (ok bool) {
	ok = atomic.LoadInt64(&f.value) == 1
	return
}

func (f *Flag) IsOff() (ok bool) {
	ok = atomic.LoadInt64(&f.value) == 0
	return
}

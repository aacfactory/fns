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

package switchs

import (
	"sync/atomic"
)

type Switch struct {
	value uint64
}

func (s *Switch) On() {
	atomic.StoreUint64(&s.value, 2)
}

func (s *Switch) Off() {
	atomic.StoreUint64(&s.value, 1)
}

func (s *Switch) Confirm() {
	n := atomic.LoadUint64(&s.value)
	switch n {
	case 2:
		atomic.StoreUint64(&s.value, 3)
		break
	case 1:
		atomic.StoreUint64(&s.value, 0)
		break
	default:
		break
	}
}

func (s *Switch) IsOn() (ok bool, confirmed bool) {
	n := atomic.LoadUint64(&s.value)
	switch n {
	case 2:
		ok = true
		break
	case 3:
		ok = true
		confirmed = true
		break
	default:
		break
	}
	return
}

func (s *Switch) IsOff() (ok bool, confirmed bool) {
	n := atomic.LoadUint64(&s.value)
	switch n {
	case 1:
		ok = true
		break
	case 0:
		ok = true
		confirmed = true
		break
	default:
		break
	}
	return
}

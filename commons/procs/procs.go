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

package procs

import (
	"go.uber.org/automaxprocs/maxprocs"
	"runtime"
)

func New(min int, max int) *AutoMaxProcs {
	if min < 0 {
		min = 0
	}
	if max < 0 {
		max = 0
	}
	return &AutoMaxProcs{
		min:     min,
		max:     max,
		resetFn: nil,
	}
}

type AutoMaxProcs struct {
	min     int
	max     int
	resetFn func()
}

func (p *AutoMaxProcs) Enable() {
	if p.min > 0 {
		reset, setErr := maxprocs.Set(maxprocs.Min(p.min), maxprocs.Logger(func(s string, i ...interface{}) {}))
		if setErr != nil {
			runtime.GOMAXPROCS(0)
			return
		}
		p.resetFn = reset
	}
	runtime.GOMAXPROCS(p.max)
	return
}

func (p *AutoMaxProcs) Reset() {
	if p.resetFn != nil {
		p.resetFn()
	}
}

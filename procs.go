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

package fns

import (
	"github.com/aacfactory/logs"
	"go.uber.org/automaxprocs/maxprocs"
	"runtime"
)

type procsOption struct {
	min int
	max int
}

func newPROCS(env Environments, opt *procsOption) *procs {
	min := opt.min
	if min < 0 {
		min = 0
	}
	max := opt.max
	if max < 0 {
		max = 0
	}
	return &procs{
		log:     env.Log().With("system", "maxprocs"),
		min:     min,
		max:     max,
		resetFn: nil,
	}
}

type procs struct {
	log     logs.Logger
	min     int
	max     int
	resetFn func()
}

func (p *procs) enable() {
	if p.min > 0 {
		maxprocsLog := &printf{
			core: p.log,
		}
		reset, setErr := maxprocs.Set(maxprocs.Min(p.min), maxprocs.Logger(maxprocsLog.Printf))
		if setErr != nil {
			if p.log.DebugEnabled() {
				p.log.Debug().Message("fns: set automaxprocs failed, use runtime.GOMAXPROCS(0) insteadof")
			}
			runtime.GOMAXPROCS(0)
			return
		}
		p.resetFn = reset
	}
	runtime.GOMAXPROCS(p.max)
	return
}

func (p *procs) reset() {
	if p.resetFn != nil {
		p.resetFn()
	}
}
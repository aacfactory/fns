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
	"github.com/aacfactory/logs"
	"go.uber.org/automaxprocs/maxprocs"
	"runtime"
)

type Options struct {
	Log logs.Logger
	Min int
	Max int
}

func New(options Options) *AutoMaxProcs {
	min := options.Min
	if min < 0 {
		min = 0
	}
	max := options.Max
	if max < 0 {
		max = 0
	}
	return &AutoMaxProcs{
		log:     options.Log.With("fns", "automaxprocs"),
		min:     min,
		max:     max,
		resetFn: nil,
	}
}

type AutoMaxProcs struct {
	log     logs.Logger
	min     int
	max     int
	resetFn func()
}

func (p *AutoMaxProcs) Enable() {
	if p.min > 0 {
		var log func(string, ...interface{})
		if p.log.DebugEnabled() {
			log = logs.MapToLogger(p.log, logs.DebugLevel, true).Printf
		} else if p.log.InfoEnabled() {
			log = logs.MapToLogger(p.log, logs.InfoLevel, false).Printf
		} else if p.log.WarnEnabled() {
			log = logs.MapToLogger(p.log, logs.WarnLevel, false).Printf
		} else if p.log.ErrorEnabled() {
			log = logs.MapToLogger(p.log, logs.ErrorLevel, false).Printf
		} else {
			log = logs.MapToLogger(p.log, logs.InfoLevel, false).Printf
		}
		reset, setErr := maxprocs.Set(maxprocs.Min(p.min), maxprocs.Logger(log))
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

func (p *AutoMaxProcs) Reset() {
	if p.resetFn != nil {
		p.resetFn()
	}
}

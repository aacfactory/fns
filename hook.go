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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"sync"
	"time"
)

type HookUnit struct {
	mutex       sync.Mutex
	Context     Context
	confirmed   bool
	counter     *sync.WaitGroup
	Service     string
	Fn          string
	Result      []byte
	Failed      bool
	FailedCause errors.CodeError
	Latency     time.Duration
}

func newHookUnit(ctx Context, service string, fn string, result []byte, failedCause errors.CodeError, latency time.Duration) *HookUnit {
	return &HookUnit{
		mutex:       sync.Mutex{},
		Context:     ctx,
		confirmed:   false,
		counter:     nil,
		Service:     service,
		Fn:          fn,
		Result:      result,
		Failed:      failedCause != nil,
		FailedCause: failedCause,
		Latency:     latency,
	}
}

func (unit *HookUnit) Confirm() {
	unit.mutex.Lock()
	defer unit.mutex.Unlock()
	if unit.confirmed {
		return
	}
	unit.counter.Done()
	unit.confirmed = true
}

type Hook interface {
	Build(env Environments) (err error)
	Handle(unit *HookUnit)
	Close()
}

func newHooks(env Environments, items []Hook) (v *hooks, err error) {
	if items == nil {
		items = make([]Hook, 0, 1)
	}
	builds := make([]Hook, 0, 1)
	for _, hook := range items {
		buildErr := hook.Build(env)
		if buildErr != nil {
			err = buildErr
			break
		}
		builds = append(builds, hook)
	}
	if err != nil {
		for _, build := range builds {
			build.Close()
		}
		err = fmt.Errorf("create hooks handler failed for build hook, %v", err)
		return
	}
	v = &hooks{
		empty:   len(builds) == 0,
		hooks:   builds,
		hookCh:  make(chan *HookUnit, 1024),
		counter: &sync.WaitGroup{},
	}
	return
}

type hooks struct {
	running *commons.SafeFlag
	empty   bool
	hooks   []Hook
	hookCh  chan *HookUnit
	counter *sync.WaitGroup
}

func (h *hooks) send(unit *HookUnit) {
	if h.empty {
		return
	}
	if h.running.IsOff() {
		return
	}
	h.counter.Add(1)
	unit.counter = h.counter
	h.hookCh <- unit
}

func (h *hooks) start() {
	h.running.On()
	go func(h *hooks) {
		for {
			unit, ok := <-h.hookCh
			if !ok {
				break
			}
			for _, hook := range h.hooks {
				hook.Handle(unit)
			}
		}
	}(h)
	return
}

func (h *hooks) close() (err error) {
	h.running.Off()
	h.counter.Wait()
	close(h.hookCh)
	for _, hook := range h.hooks {
		hook.Close()
	}
	return
}

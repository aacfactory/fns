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
	"strings"
)

type FnBus interface {
	Request(ctx Context, namespace string, name string, arg Argument) (result Result)
	Handle(namespace string, handle FnRequestHandle)
	Start()
	Close()
}

func newFnsBus(concurrency int, maxIdleTimeSecond int, discovery Discovery) (bus FnBus) {
	var wup workUnitHandler
	if discovery == nil {
		wup = newStandaloneWorkUnitHandler()
	} else {
		wup = newClusterWorkUnitHandler(discovery)
	}
	wp := newWorkers(WorkersConfig{
		Concurrency:       concurrency,
		MaxIdleTimeSecond: maxIdleTimeSecond,
	}, wup)
	bus = &fnBus{
		namespaces: make([]string, 0, 1),
		wp:         wp,
	}
	return
}

type fnBus struct {
	namespaces []string
	wp         *workers
}

func (bus *fnBus) Request(ctx Context, namespace string, name string, arg Argument) (result Result) {
	requestCtx := ctx.WithFnFork(namespace, name)
	result = NewResult()
	ok := bus.wp.Execute(requestCtx, arg, result)
	if !ok {
		result.Set(TimeoutError(fmt.Sprintf("execute %s %s timeout", namespace, name)))
		return
	}
	return
}

func (bus *fnBus) Handle(namespace string, handle FnRequestHandle) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" || handle == nil {
		return
	}
	bus.wp.unitHandler.Bind(namespace, handle)
	bus.namespaces = append(bus.namespaces)
}

func (bus *fnBus) Start() {
	bus.wp.Start()
}

func (bus *fnBus) Close() {
	bus.wp.Stop()
	bus.wp.Sync()
	for _, ns := range bus.namespaces {
		bus.wp.unitHandler.UnBind(ns)
	}
}

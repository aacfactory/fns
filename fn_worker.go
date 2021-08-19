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
	"github.com/dgraph-io/ristretto"
	"runtime"
	"sync"
	"time"
)

func initFnWorkers() {
	if runtime.GOMAXPROCS(0) == 1 {
		workerChanCap = 0
	}
	workerChanCap = 1
	cpuNum := runtime.NumCPU()
	workersConcurrency = cpuNum * 32 * 1024
}

var workerChanCap int
var workersConcurrency int

type workUnit struct {
	ctx    Context
	arg    Argument
	result Result
}

type workUnitHandler interface {
	Handle(unit *workUnit)
	Bind(namespace string, handler FnRequestHandle) (ok bool)
	UnBind(namespace string)
	Exist(namespace string) (has bool)
}

func newStandaloneWorkUnitHandler() (wuh workUnitHandler) {
	wuh = &standaloneWorkUnitHandler{
		handles: make(map[string]FnRequestHandle),
	}
	return
}

type standaloneWorkUnitHandler struct {
	handles map[string]FnRequestHandle
}

func (h *standaloneWorkUnitHandler) Bind(namespace string, handler FnRequestHandle) (ok bool) {
	if namespace == "" || handler == nil || h.Exist(namespace) {
		return
	}
	h.handles[namespace] = handler
	ok = true
	return
}

func (h *standaloneWorkUnitHandler) UnBind(namespace string) {
	if h.Exist(namespace) {
		delete(h.handles, namespace)
	}
	return
}

func (h *standaloneWorkUnitHandler) Exist(namespace string) (has bool) {
	_, has = h.handles[namespace]
	return
}

func (h *standaloneWorkUnitHandler) Handle(unit *workUnit) {
	ctx := unit.ctx
	arg := unit.arg
	result := unit.result
	namespace := ctx.Meta().Namespace()
	handler, has := h.handles[namespace]
	if !has {
		fnName := ctx.Meta().FnName()
		result.Set(NotFoundError(fmt.Sprintf("%s in %s was not found", fnName, namespace)))
		return
	}
	if ctx.Timeout() {
		fnName := ctx.Meta().FnName()
		result.Set(TimeoutError(fmt.Sprintf("%s in %s was timeout", fnName, namespace)))
		return
	}
	v, err := handler(ctx, arg)
	if err != nil {
		result.Set(err)
	} else {
		result.Set(v)
	}
}

func newClusterWorkUnitHandler(discovery Discovery) (wuh workUnitHandler) {
	cache, newCacheErr := ristretto.NewCache(&ristretto.Config{
		NumCounters: 128 * (1 << 20) / 100, // number of keys to track frequency of (10M).
		MaxCost:     128 * (1 << 20),       // maximum cost of cache (1GB).
		BufferItems: 64,                    // number of keys per Get buffer.
	})
	if newCacheErr != nil {
		panic(fmt.Errorf("new cluster WorkUnitHandler failed, %v", newCacheErr))
		return
	}
	wuh0 := &clusterWorkUnitHandler{
		standalone: standaloneWorkUnitHandler{
			handles: make(map[string]FnRequestHandle),
		},
		discovery:       discovery,
		registrationIds: make(map[string]string),
		remoteClients:   cache,
	}
	ch, syncErr := discovery.SyncRegistrations()
	if syncErr == nil {
		wuh0.listen(ch)
	}
	wuh = wuh0
	return
}

type clusterWorkUnitHandler struct {
	standalone      standaloneWorkUnitHandler
	discovery       Discovery
	registrationIds map[string]string
	remoteClients   *ristretto.Cache
}

func (h *clusterWorkUnitHandler) listen(ch <-chan map[string][]Registration) {
	go func(h *clusterWorkUnitHandler, ch <-chan map[string][]Registration) {
		for {
			registrationMap, ok := <-ch
			if !ok {
				break
			}
			for name, registrations := range registrationMap {
				if registrations == nil || len(registrations) == 0 {
					h.remoteClients.Del(name)
					h.remoteClients.Wait()
					continue
				}
				remoteClient0, remoteClientErr := discoveryRegistrationMapToHttpClient(registrations)
				if remoteClientErr != nil {
					h.remoteClients.Del(name)
					h.remoteClients.Wait()
					continue
				}
				h.remoteClients.SetWithTTL(name, remoteClient0, 1, 10*time.Minute)
				h.remoteClients.Wait()
			}
		}
	}(h, ch)
}

func (h *clusterWorkUnitHandler) Handle(unit *workUnit) {
	ctx := unit.ctx
	namespace := ctx.Meta().Namespace()
	if h.Exist(namespace) {
		h.standalone.Handle(unit)
		return
	}
	arg := unit.arg
	result := unit.result
	fnName := ctx.Meta().FnName()
	// remote
	var remoteClient FnHttpClient
	cached, hasCached := h.remoteClients.Get(namespace)
	if !hasCached {
		registrations, getErr := h.discovery.Get(namespace)
		if getErr != nil {
			result.Set(getErr)
			return
		}
		if registrations == nil || len(registrations) == 0 {
			result.Set(NotFoundError(fmt.Sprintf("%s in %s was not found", fnName, namespace)))
			return
		}
		remoteClient0, remoteClientErr := discoveryRegistrationMapToHttpClient(registrations)
		if remoteClientErr != nil {
			result.Set(remoteClientErr)
			return
		}
		h.remoteClients.SetWithTTL(namespace, remoteClient0, 1, 10*time.Minute)
		h.remoteClients.Wait()
		remoteClient = remoteClient0
	} else {
		remoteClient, _ = cached.(FnHttpClient)
	}
	response, remoteErr := remoteClient.Request(ctx, arg)
	if remoteErr != nil {
		result.Set(remoteErr)
		return
	}
	result.Set(response)
}

func (h *clusterWorkUnitHandler) Bind(namespace string, handler FnRequestHandle) (ok bool) {
	ok = h.standalone.Bind(namespace, handler)
	if !ok {
		return
	}
	registrationId, publishErr := h.discovery.Publish(namespace)
	if publishErr != nil {
		h.standalone.UnBind(namespace)
		ok = false
		return
	}
	h.registrationIds[namespace] = registrationId
	return
}

func (h *clusterWorkUnitHandler) UnBind(namespace string) {
	registrationId, has := h.registrationIds[namespace]
	if !has {
		return
	}
	_ = h.discovery.UnPublish(registrationId)
	h.standalone.UnBind(namespace)
	return
}

func (h *clusterWorkUnitHandler) Exist(namespace string) (has bool) {
	has = h.standalone.Exist(namespace)
	return
}

type workerChan struct {
	lastUseTime time.Time
	ch          chan *workUnit
}

type WorkersConfig struct {
	Concurrency       int `json:"concurrency,omitempty"`
	MaxIdleTimeSecond int `json:"maxIdleTimeSecond,omitempty"`
}

func newWorkers(config WorkersConfig, unitHandler workUnitHandler) (w *workers) {
	concurrency := config.Concurrency
	if concurrency < 1 {
		concurrency = workersConcurrency
	}
	maxIdleTime := config.MaxIdleTimeSecond
	if maxIdleTime <= 0 {
		maxIdleTime = 10
	}
	if unitHandler == nil {
		panic("create workers failed cause workUnitHandler is nil")
		return
	}
	w = &workers{
		unitHandler:           unitHandler,
		maxWorkersCount:       concurrency,
		maxIdleWorkerDuration: time.Duration(maxIdleTime) * time.Second,
		lock:                  sync.Mutex{},
		workersCount:          0,
		mustStop:              false,
		workerChanPool:        sync.Pool{},
		unitPool:              sync.Pool{},
		wg:                    sync.WaitGroup{},
	}
	return
}

type workers struct {
	maxWorkersCount       int
	maxIdleWorkerDuration time.Duration
	lock                  sync.Mutex
	workersCount          int
	mustStop              bool
	ready                 []*workerChan
	stopCh                chan struct{}
	workerChanPool        sync.Pool
	unitPool              sync.Pool
	unitHandler           workUnitHandler
	wg                    sync.WaitGroup
}

func (w *workers) Execute(ctx Context, arg Argument, result Result) (ok bool) {
	ch := w.getCh()
	if ch == nil {
		return false
	}
	unit := w.unitPool.Get().(*workUnit)
	unit.ctx = ctx
	unit.arg = arg
	unit.result = result
	w.wg.Add(1)
	ch.ch <- unit
	return true
}

func (w *workers) Start() {
	if w.stopCh != nil {
		panic("workers is already started")
	}
	w.stopCh = make(chan struct{})
	stopCh := w.stopCh
	w.workerChanPool.New = func() interface{} {
		return &workerChan{
			ch: make(chan *workUnit, workerChanCap),
		}
	}
	w.unitPool.New = func() interface{} {
		return &workUnit{
			ctx:    nil,
			arg:    nil,
			result: nil,
		}
	}
	go func() {
		var scratch []*workerChan
		for {
			w.clean(&scratch)
			select {
			case <-stopCh:
				return
			default:
				time.Sleep(w.maxIdleWorkerDuration)
			}
		}
	}()
}

func (w *workers) Stop() {
	if w.stopCh == nil {
		panic("workers wasn't started")
	}
	close(w.stopCh)
	w.stopCh = nil
	w.lock.Lock()
	ready := w.ready
	for i := range ready {
		ready[i].ch <- nil
		ready[i] = nil
	}
	w.ready = ready[:0]
	w.mustStop = true
	w.lock.Unlock()
}

func (w *workers) Sync() {
	w.wg.Wait()
}

func (w *workers) clean(scratch *[]*workerChan) {
	maxIdleWorkerDuration := w.maxIdleWorkerDuration

	criticalTime := time.Now().Add(-maxIdleWorkerDuration)

	w.lock.Lock()
	ready := w.ready
	n := len(ready)

	l, r, mid := 0, n-1, 0
	for l <= r {
		mid = (l + r) / 2
		if criticalTime.After(w.ready[mid].lastUseTime) {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	i := r
	if i == -1 {
		w.lock.Unlock()
		return
	}

	*scratch = append((*scratch)[:0], ready[:i+1]...)
	m := copy(ready, ready[i+1:])
	for i = m; i < n; i++ {
		ready[i] = nil
	}
	w.ready = ready[:m]
	w.lock.Unlock()

	tmp := *scratch
	for i := range tmp {
		tmp[i].ch <- nil
		tmp[i] = nil
	}
}

func (w *workers) getCh() *workerChan {
	var ch *workerChan
	createWorker := false

	w.lock.Lock()
	ready := w.ready
	n := len(ready) - 1
	if n < 0 {
		if w.workersCount < w.maxWorkersCount {
			createWorker = true
			w.workersCount++
		}
	} else {
		ch = ready[n]
		ready[n] = nil
		w.ready = ready[:n]
	}
	w.lock.Unlock()

	if ch == nil {
		if !createWorker {
			return nil
		}
		vch := w.workerChanPool.Get()
		ch = vch.(*workerChan)
		go func() {
			w.handle(ch)
			w.workerChanPool.Put(vch)
		}()
	}
	return ch
}

func (w *workers) release(ch *workerChan) bool {
	ch.lastUseTime = time.Now()
	w.lock.Lock()
	if w.mustStop {
		w.lock.Unlock()
		return false
	}
	w.ready = append(w.ready, ch)
	w.lock.Unlock()
	return true
}

func (w *workers) handle(ch *workerChan) {
	var unit *workUnit

	for unit = range ch.ch {
		if unit == nil {
			break
		}
		w.unitHandler.Handle(unit)
		w.wg.Done()
		w.unitPool.Put(unit)
		if !w.release(ch) {
			break
		}
	}

	w.lock.Lock()
	w.workersCount--
	w.lock.Unlock()
}

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

package cluster

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/container/ring"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Registration struct {
	Id               string
	Name             string
	Internal         bool
	Address          string
	client           Client
	unavailableTimes int64
	checkLock        sync.Mutex
	lastCheckTime    time.Time
}

func (r *Registration) Key() (key string) {
	key = r.Id
	return
}

func (r *Registration) Request(ctx context.Context, fn string, argument service.Argument) (result service.Result) {
	req, hasReq := service.GetRequest(ctx)
	if !hasReq {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: remote call failed, there is no request in context").WithMeta("service", r.Name).WithMeta("fn", fn)))
		return
	}
	user, userErr := json.Marshal(req.User())
	if userErr != nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: remote call failed, encode request user failed").WithCause(userErr).WithMeta("service", r.Name).WithMeta("fn", fn)))
		return
	}
	arg, argErr := argument.MarshalJSON()
	if argErr != nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: remote call failed, encode argument failed").WithCause(argErr).WithMeta("service", r.Name).WithMeta("fn", fn)))
		return
	}
	ir := &internalRequest{
		User:     user,
		Argument: arg,
	}
	reqBody, encodeErr := json.Marshal(ir)
	if encodeErr != nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: remote call failed, encode internal request body failed").WithCause(encodeErr).WithMeta("service", r.Name).WithMeta("fn", fn)))
		return
	}
	reqBody = encodeRequestBody(reqBody)
	reqHeader := req.Header().Raw().Clone()
	reqHeader.Set(httpContentType, httpContentTypeProxy)
	fr := service.NewResult()
	result = fr
	status, _, respBody, callErr := r.client.Do(ctx, http.MethodPost, r.Address, fmt.Sprintf("/%s/%s", r.Name, fn), reqHeader, reqBody)
	if callErr != nil {
		r.addUnavailableTimes()
		fr.Failed(errors.Warning("fns: remote call failed").WithCause(callErr).WithMeta("service", r.Name).WithMeta("fn", fn))
		return
	}
	resp := &response{}
	decodeErr := json.Unmarshal(respBody, resp)
	if decodeErr != nil {
		fr.Failed(errors.Warning("fns: remote call failed, response body is not json").WithCause(callErr).WithMeta("service", r.Name).WithMeta("fn", fn))
		return
	}
	tracer, hasTracer := service.GetTracer(ctx)
	if hasTracer && resp.HasSpan() {
		span, _ := resp.Span()
		if span != nil {
			tracer.Span().AppendChild(span)
		}
	}
	if status == http.StatusOK {
		fr.Succeed(resp.Data)
	} else {
		fr.Failed(resp.AsError())
	}
	return
}

func (r *Registration) available() (ok bool) {
	status, _, body, callErr := r.client.Do(context.TODO(), http.MethodGet, r.Address, "/health", nil, nil)
	if callErr != nil {
		return
	}
	if status != http.StatusOK {
		return
	}
	if body == nil || !json.Validate(body) {
		return
	}
	obj := json.NewObjectFromBytes(body)
	_ = obj.Get("running", &ok)
	return
}

func (r *Registration) addUnavailableTimes() {
	atomic.AddInt64(&r.unavailableTimes, 1)
	return
}

func (r *Registration) resetUnavailableTimes() {
	atomic.StoreInt64(&r.unavailableTimes, 0)
	return
}

func (r *Registration) unavailable() (ok bool) {
	ok = atomic.LoadInt64(&r.unavailableTimes) > 5
	if ok {
		r.checkLock.Lock()
		if time.Now().Sub(r.lastCheckTime) < 10*time.Second {
			r.checkLock.Unlock()
			return
		}
		r.lastCheckTime = time.Now()
		r.checkLock.Unlock()
		if r.available() {
			r.resetUnavailableTimes()
		}
	}
	return
}

func newRegistrations(value *Registration) *Registrations {
	return &Registrations{
		r: ring.New(value),
	}
}

type Registrations struct {
	r *ring.Ring
}

func (r *Registrations) Next() (v *Registration, has bool) {
	p := r.r.Next()
	if p == nil {
		return
	}
	v, has = p.(*Registration)
	return
}

func (r *Registrations) Append(v *Registration) {
	r.r.Append(v)
	return
}

func (r *Registrations) Remove(v *Registration) {
	r.r.Remove(v)
	return
}

func (r *Registrations) Size() (size int) {
	size = r.r.Size()
	return
}

func (r *Registrations) Get(id string) (v *Registration, has bool) {
	if id == "" {
		return
	}
	p := r.r.Get(id)
	if p == nil {
		return
	}
	v, has = p.(*Registration)
	return
}

func newRegistrationsManager(log logs.Logger) *RegistrationsManager {
	return &RegistrationsManager{
		log:    log.With("cluster", "registrations"),
		stopCh: make(chan struct{}, 1),
		events: make(chan *nodeEvent, 512),
		nodes:  sync.Map{},
		values: sync.Map{},
	}
}

type RegistrationsManager struct {
	log    logs.Logger
	stopCh chan struct{}
	events chan *nodeEvent
	nodes  sync.Map
	values sync.Map
}

func (manager *RegistrationsManager) members() (values []*node) {
	values = make([]*node, 0, 1)
	manager.nodes.Range(func(_, value interface{}) bool {
		n := value.(*node)
		values = append(values, n)
		return true
	})
	return
}

func (manager *RegistrationsManager) containsNode(node *node) (ok bool) {
	_, ok = manager.nodes.Load(node.Id_)
	return
}

func (manager *RegistrationsManager) register(n *node) {
	if manager.containsNode(n) {
		return
	}
	manager.events <- &nodeEvent{
		kind:  "register",
		value: n,
	}
	return
}

func (manager *RegistrationsManager) deregister(n *node) {
	if !manager.containsNode(n) {
		return
	}
	manager.events <- &nodeEvent{
		kind:  "deregister",
		value: n,
	}
	return
}

func (manager *RegistrationsManager) handleRegister(node *node) {
	if manager.containsNode(node) {
		return
	}
	manager.nodes.Store(node.Id_, node)
	registrations := node.registrations()
	for _, registration := range registrations {
		value, has := manager.values.Load(registration.Name)
		if has {
			registered := value.(*Registrations)
			_, exist := registered.Get(registration.Id)
			if exist {
				continue
			}
			registered.Append(registration)
		} else {
			manager.values.Store(registration.Name, newRegistrations(registration))
		}
	}
	return
}

func (manager *RegistrationsManager) handleDeregister(n *node) {
	existNode0, hasNode := manager.nodes.Load(n.Id_)
	if !hasNode {
		return
	}
	manager.nodes.Delete(n.Id_)
	existNode := existNode0.(*node)
	registrations := existNode.registrations()
	for _, registration := range registrations {
		value, has := manager.values.Load(registration.Name)
		if !has {
			continue
		}
		registered := value.(*Registrations)
		existed, exist := registered.Get(registration.Id)
		if !exist {
			continue
		}
		registered.Remove(existed)
		if registered.Size() == 0 {
			manager.values.Delete(registration.Name)
		}
	}
	return
}

func (manager *RegistrationsManager) listenEvents() {
	go func() {
		closed := false
		for {
			if closed {
				break
			}
			select {
			case event, ok := <-manager.events:
				if !ok {
					closed = true
					break
				}
				if event.kind == "register" {
					manager.handleRegister(event.value)
				} else if event.kind == "deregister" {
					manager.handleDeregister(event.value)
				}
			case <-manager.stopCh:
				closed = true
				break
			}
		}
	}()
}

func (manager *RegistrationsManager) removeUnavailableNodes() {
	unavailableNodes := make([]*node, 0, 1)
	manager.nodes.Range(func(_, value interface{}) bool {
		n := value.(*node)
		if n.available() {
			return true
		}
		unavailableNodes = append(unavailableNodes, n)
		return true
	})
	for _, unavailable := range unavailableNodes {
		manager.deregister(unavailable)
	}
}

func (manager *RegistrationsManager) Close() {
	close(manager.stopCh)
}

func (manager *RegistrationsManager) Get(_ context.Context, name string) (endpoint service.Endpoint, has bool) {
	value, exist := manager.values.Load(name)
	if !exist {
		return
	}
	registered := value.(*Registrations)
	for i := 0; i < 5; i++ {
		registration, ok := registered.Next()
		if !ok {
			return
		}
		if registration.unavailable() {
			continue
		}
		endpoint = registration
		has = true
		return
	}
	return
}

func (manager *RegistrationsManager) GetExact(_ context.Context, name string, id string) (endpoint service.Endpoint, has bool) {
	value, exist := manager.values.Load(name)
	if !exist {
		return
	}
	registered := value.(*Registrations)
	for {
		registration, ok := registered.Next()
		if !ok {
			return
		}
		if registration.Id == id {
			if registration.unavailable() {
				return
			}
			endpoint = registration
			has = true
			return
		}
	}
}

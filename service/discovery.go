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

package service

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/rings"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type EndpointDiscoveryGetOption func(options *EndpointDiscoveryGetOptions)

type EndpointDiscoveryGetOptions struct {
	id           string
	native       bool
	versionRange []versions.Version
}

func Exact(id string) EndpointDiscoveryGetOption {
	return func(options *EndpointDiscoveryGetOptions) {
		options.id = strings.TrimSpace(id)
		return
	}
}

func Native() EndpointDiscoveryGetOption {
	return func(options *EndpointDiscoveryGetOptions) {
		options.native = true
		return
	}
}

func VersionRange(left versions.Version, right versions.Version) EndpointDiscoveryGetOption {
	return func(options *EndpointDiscoveryGetOptions) {
		options.versionRange[0] = left
		options.versionRange[1] = right
		return
	}
}

type EndpointDiscovery interface {
	Get(ctx context.Context, service string, options ...EndpointDiscoveryGetOption) (endpoint Endpoint, has bool)
}

func newRegistrationTask(registration *Registration, handleTimeout time.Duration, hook func(task *registrationTask)) *registrationTask {
	return &registrationTask{
		registration: registration,
		r:            nil,
		result:       nil,
		timeout:      handleTimeout,
		hook:         hook,
	}
}

type registrationTask struct {
	registration *Registration
	r            Request
	result       ResultWriter
	timeout      time.Duration
	hook         func(task *registrationTask)
}

func (task *registrationTask) begin(r Request, w ResultWriter) {
	task.r = r
	task.result = w
}

func (task *registrationTask) end() {
	task.r = nil
	task.result = nil
}

func (task *registrationTask) Execute(ctx context.Context) {
	defer task.hook(task)

	timeout := task.timeout
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		timeout = deadline.Sub(time.Now())
	}

	registration := task.registration
	r := task.r
	future := task.result

	var body json.RawMessage
	if r.Argument() != nil {
		var bodyErr error
		body, bodyErr = json.Marshal(r.Argument())
		if bodyErr != nil {
			future.Failed(errors.Warning("fns: registration request failed").WithCause(bodyErr))
			return
		}
	}

	requestBody, encodeErr := json.Marshal(internalRequest{
		User:  r.User(),
		Trunk: r.Trunk(),
		Body:  body,
	})
	if encodeErr != nil {
		future.Failed(errors.Warning("fns: registration request failed").WithCause(encodeErr))
		return
	}
	header := r.Header().MapToHttpHeader()
	header.Del(httpRequestVersionsHeader)
	header.Add(httpRequestInternalHeader, "true")
	header.Add(httpRequestIdHeader, r.Id())
	header.Add(httpRequestSignatureHeader, bytex.ToString(registration.signer.Sign(requestBody)))
	header.Add(httpRequestTimeoutHeader, fmt.Sprintf("%d", uint64(timeout/time.Millisecond)))
	serviceName, fn := r.Fn()
	status, _, responseBody, postErr := registration.client.Post(ctx, fmt.Sprintf("/%s/%s", serviceName, fn), header, requestBody)
	if postErr != nil {
		future.Failed(errors.Warning("fns: registration request failed").WithCause(postErr))
		return
	}
	ir := &internalResponseImpl{}
	decodeErr := json.Unmarshal(responseBody, ir)
	if decodeErr != nil {
		future.Failed(errors.Warning("fns: registration request failed").WithCause(decodeErr))
		return
	}

	// Span
	trace, hasTracer := GetTracer(ctx)
	if hasTracer && ir.Span != nil {
		trace.Span().AppendChild(ir.Span)
	}
	// user
	if ir.User != nil {
		if !r.User().Authenticated() && ir.User.Authenticated() {
			r.User().SetId(ir.User.Id())
		}
		if r.User().Authenticated() && ir.User.Authenticated() {
			r.User().SetAttributes(ir.User.Attributes())
		}
	}
	// trunk
	if ir.Trunk != nil {
		r.Trunk().ReadFrom(ir.Trunk)
	}
	// body
	if status == http.StatusOK {
		future.Succeed(ir.Body)
	} else {
		future.Failed(errors.Decode(ir.Body))
	}
	return
}

type Registration struct {
	hostId  string
	id      string
	version versions.Version
	address string
	name    string
	client  HttpClient
	signer  *secret.Signer
	worker  Workers
	timeout time.Duration
	pool    sync.Pool
}

func (registration *Registration) Key() (key string) {
	key = registration.id
	return
}

func (registration *Registration) Name() (name string) {
	name = registration.name
	return
}

func (registration *Registration) Internal() (ok bool) {
	ok = true
	return
}

func (registration *Registration) Document() (document Document) {
	return
}

func (registration *Registration) Request(ctx context.Context, r Request) (result Result) {
	future := NewResult()
	task := registration.acquire()
	task.begin(r, future)
	if !registration.worker.Dispatch(ctx, task) {
		future.Failed(errors.Timeout("fns: endpoint execute timeout"))
		registration.release(task)
	}
	result = future
	return
}

func (registration *Registration) RequestSync(ctx context.Context, r Request) (result interface{}, has bool, err errors.CodeError) {
	future := registration.Request(ctx, r)
	result, has, err = future.Value(ctx)
	return
}

func (registration *Registration) acquire() (task *registrationTask) {
	v := registration.pool.Get()
	if v != nil {
		task = v.(*registrationTask)
		return
	}
	task = newRegistrationTask(registration, registration.timeout, func(task *registrationTask) {
		registration.release(task)
	})
	return
}

func (registration *Registration) release(task *registrationTask) {
	task.end()
	registration.pool.Put(task)
	return
}

type Registrations struct {
	id      string
	locker  sync.Mutex
	values  sync.Map
	signer  *secret.Signer
	dialer  HttpClientDialer
	worker  Workers
	timeout time.Duration
}

func (r *Registrations) Add(registration *Registration) {
	var ring *rings.Ring[*Registration]
	v, loaded := r.values.Load(registration.name)
	if !loaded {
		v = rings.New[*Registration](registration.name)
		r.values.Store(registration.name, v)
	}
	ring, _ = v.(*rings.Ring[*Registration])
	r.locker.Lock()
	ring.Push(registration)
	r.locker.Unlock()
	return
}

func (r *Registrations) Remove(id string) {
	r.values.Range(func(key, value any) bool {
		ring, _ := value.(*rings.Ring[*Registration])
		_, has := ring.Get(id)
		if has {
			r.locker.Lock()
			ring.Remove(id)
			r.locker.Unlock()
		}
		return true
	})
	return
}

func (r *Registrations) GetExact(name string, id string) (registration *Registration, has bool) {
	v, loaded := r.values.Load(name)
	if !loaded {
		return
	}
	ring, _ := v.(*rings.Ring[*Registration])
	registration, has = ring.Get(id)
	if !has || registration == nil {
		return
	}
	return
}

func (r *Registrations) Get(name string, vrb versions.Version, vre versions.Version) (registration *Registration, has bool) {
	v, loaded := r.values.Load(name)
	if !loaded {
		return
	}
	ring, _ := v.(*rings.Ring[*Registration])
	if ring.Len() == 0 {
		return
	}
	size := ring.Len()
	for i := 0; i < size; i++ {
		registration = ring.Next()
		if registration == nil {
			return
		}
		if registration.version.Between(vrb, vre) {
			has = true
			return
		}
	}
	return
}

func (r *Registrations) Close() {
	r.locker.Lock()
	r.values.Range(func(key, value any) bool {
		entries := value.(*rings.Ring[*Registration])
		size := entries.Len()
		for i := 0; i < size; i++ {
			entry, ok := entries.Pop()
			if ok {
				entry.client.Close()
			}
		}
		return true
	})
	r.locker.Unlock()
	return
}

func (r *Registrations) Ids() (ids []string) {
	ids = make([]string, 0, 1)
	r.values.Range(func(key, value any) bool {
		ring, _ := value.(*rings.Ring[*Registration])
		r.locker.Lock()
		size := ring.Len()
		for i := 0; i < size; i++ {
			registration := ring.Next()
			if registration == nil {
				continue
			}
			id := registration.id
			if len(ids) == 0 {
				ids = append(ids, id)
				continue
			}
			if sort.SearchStrings(ids, id) < len(ids) {
				continue
			}
			ids = append(ids, id)
			sort.Strings(ids)
		}
		r.locker.Unlock()
		return true
	})
	return
}

func (r *Registrations) AddNode(node Node) (err error) {
	address := strings.TrimSpace(node.Address)
	if address == "" {
		return
	}
	client, dialErr := r.dialer.Dial(address)
	if dialErr != nil {
		err = errors.Warning("fns: registrations dial node failed").WithCause(dialErr).WithMeta("address", address)
		return
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	header := http.Header{}
	header.Add(httpDeviceIdHeader, r.id)
	header.Add(httpRequestSignatureHeader, bytex.ToString(r.signer.Sign(bytex.FromString(r.id))))
	status, _, responseBody, getErr := client.Get(ctx, "/services/names", header)
	if getErr != nil {
		err = errors.Warning("fns: registrations get service names from node failed").WithCause(dialErr).WithMeta("address", address)
		return
	}
	if status != http.StatusOK {
		if len(responseBody) == 0 {
			err = errors.Warning("fns: registrations get service names from node failed").WithMeta("address", address).WithMeta("status", strconv.Itoa(status))
			return
		}
		err = errors.Decode(responseBody)
		return
	}
	names := make([]string, 0, 1)
	decodeErr := json.Unmarshal(responseBody, &names)
	if decodeErr != nil {
		err = errors.Warning("fns: registrations get service names from node failed").WithMeta("address", address).WithCause(decodeErr)
		return
	}
	if len(names) == 0 {
		r.Remove(node.Id)
		return
	}
	for _, name := range names {
		registration := &Registration{
			hostId:  r.id,
			id:      node.Name,
			version: node.Version,
			address: address,
			name:    name,
			client:  client,
			signer:  r.signer,
			worker:  r.worker,
			timeout: r.timeout,
			pool:    sync.Pool{},
		}
		registration.pool.New = func() any {
			return newRegistrationTask(registration, registration.timeout, func(task *registrationTask) {
				registration.release(task)
			})
		}
		r.Add(registration)
	}
	return
}

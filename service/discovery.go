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
	"github.com/aacfactory/fns/service/documents"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
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
	id              string
	native          bool
	requestVersions RequestVersions
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

func Versions(requestVersions RequestVersions) EndpointDiscoveryGetOption {
	return func(options *EndpointDiscoveryGetOptions) {
		options.requestVersions = requestVersions
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
	result       Promise
	timeout      time.Duration
	hook         func(task *registrationTask)
}

func (task *registrationTask) begin(r Request, w Promise) {
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
	fr := task.result

	// todo check cache control

	if r.Internal() {
		// body
		requestBody, encodeErr := json.Marshal(internalRequest{
			User:     r.User(),
			Trunk:    r.Trunk(),
			Argument: r.Argument(),
		})
		if encodeErr != nil {
			fr.Failed(errors.Warning("fns: registration request internal failed").WithCause(encodeErr))
			return
		}
		// header
		header := r.Header().Clone()
		header.SetAcceptVersions(r.AcceptedVersions())
		header.Add(httpRequestInternalHeader, "true")
		header.Add(httpRequestIdHeader, r.Id())
		header.Add(httpRequestSignatureHeader, bytex.ToString(registration.signer.Sign(requestBody)))
		header.Add(httpRequestTimeoutHeader, fmt.Sprintf("%d", uint64(timeout/time.Millisecond)))

		serviceName, fn := r.Fn()

		req, reqErr := transports.NewRequest(ctx, bytex.FromString(http.MethodPost), bytex.FromString(fmt.Sprintf("/%s/%s", serviceName, fn)))
		if reqErr != nil {
			fr.Failed(errors.Warning("fns: registration request internal failed").WithCause(reqErr))
			return
		}
		for name, vv := range header {
			for _, v := range vv {
				req.Header().Add(name, v)
			}
		}
		req.SetBody(requestBody)
		resp, postErr := registration.client.Do(ctx, req)
		if postErr != nil {
			// todo add error times in reg
			fr.Failed(errors.Warning("fns: registration request internal failed").WithCause(postErr))
			return
		}

		if resp.Header.Get(httpConnectionHeader) == httpCloseHeader {
			// todo mark reg is closed

		}
		if resp.Status != http.StatusOK {
			var body errors.CodeError
			if resp.Body == nil || len(resp.Body) == 0 {
				body = errors.Warning("nil error")
			} else {
				body = errors.Decode(resp.Body)
			}
			fr.Failed(body)
			return
		}

		// todo cache control

		ir := &internalResponseImpl{}
		decodeErr := json.Unmarshal(resp.Body, ir)
		if decodeErr != nil {
			fr.Failed(errors.Warning("fns: registration request internal failed").WithCause(decodeErr))
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
		if ir.Succeed {
			fr.Succeed(ir.Body)
		} else {
			fr.Failed(errors.Decode(ir.Body))
		}
	} else {
		requestBody, encodeErr := json.Marshal(r.Argument())
		if encodeErr != nil {
			fr.Failed(errors.Warning("fns: registration request failed").WithCause(encodeErr))
			return
		}
		header := r.Header()
		serviceName, fn := r.Fn()

		req, reqErr := transports.NewRequest(ctx, bytex.FromString(http.MethodPost), bytex.FromString(fmt.Sprintf("/%s/%s", serviceName, fn)))
		if reqErr != nil {
			fr.Failed(errors.Warning("fns: registration request failed").WithCause(reqErr))
			return
		}
		for name, vv := range header {
			for _, v := range vv {
				req.Header().Add(name, v)
			}
		}
		req.SetBody(requestBody)
		resp, postErr := registration.client.Do(ctx, req)
		if postErr != nil {
			// todo add error times in reg
			fr.Failed(errors.Warning("fns: registration request failed").WithCause(postErr))
			return
		}

		if resp.Header.Get(httpConnectionHeader) == httpCloseHeader {
			// todo mark reg is closed

		}
		if resp.Status != http.StatusOK {
			var body errors.CodeError
			if resp.Body == nil || len(resp.Body) == 0 {
				body = errors.Warning("nil error")
			} else {
				body = errors.Decode(resp.Body)
			}
			fr.Failed(body)
			return
		}
		if resp.Body == nil || len(resp.Body) == 0 {
			fr.Succeed(nil)
		} else {
			// todo cache control
			fr.Succeed(json.RawMessage(resp.Body))
		}
	}
	return
}

type RegistrationList []*Registration

func (list RegistrationList) Len() int {
	return len(list)
}

func (list RegistrationList) Less(i, j int) bool {
	return list[i].version.LessThan(list[j].version)
}

func (list RegistrationList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
	return
}

func (list RegistrationList) MinVersion() (r *Registration) {
	size := len(list)
	if size == 0 {
		return
	}
	r = list[0]
	return
}

func (list RegistrationList) MaxVersion() (r *Registration) {
	size := len(list)
	if size == 0 {
		return
	}
	r = list[size-1]
	return
}

type Registration struct {
	hostId  string
	id      string
	version versions.Version
	address string
	name    string
	client  transports.Client
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

func (registration *Registration) Document() (document *documents.Document) {
	return
}

func (registration *Registration) Request(ctx context.Context, r Request) (future Future) {
	promise, fr := NewFuture()
	task := registration.acquire()
	task.begin(r, promise)
	if !registration.worker.Dispatch(ctx, task) {
		promise.Failed(ErrServiceOverload)
		registration.release(task)
	}
	future = fr
	return
}

func (registration *Registration) RequestSync(ctx context.Context, r Request) (result FutureResult, err errors.CodeError) {
	fr := registration.Request(ctx, r)
	result, err = fr.Get(ctx)
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

func newRegistrations(log logs.Logger, id string, name string, version versions.Version, cluster Cluster, worker Workers, dialer transports.Dialer, signer *secret.Signer, timeout time.Duration, refreshInterval time.Duration) *Registrations {
	return &Registrations{
		id:              id,
		name:            name,
		version:         version,
		log:             log,
		cluster:         cluster,
		values:          sync.Map{},
		signer:          signer,
		dialer:          dialer,
		worker:          worker,
		timeout:         timeout,
		refreshInterval: refreshInterval,
	}
}

type Registrations struct {
	id              string
	name            string
	version         versions.Version
	log             logs.Logger
	cluster         Cluster
	values          sync.Map
	signer          *secret.Signer
	dialer          transports.Dialer
	worker          Workers
	timeout         time.Duration
	refreshInterval time.Duration
}

func (r *Registrations) Add(registration *Registration) {
	var ring *rings.Ring[*Registration]
	v, loaded := r.values.Load(registration.name)
	if !loaded || v == nil {
		v = rings.New[*Registration](registration.name)
		r.values.Store(registration.name, v)
	}
	ring, _ = v.(*rings.Ring[*Registration])
	ring.Push(registration)
	return
}

func (r *Registrations) Remove(id string) {
	empties := make([]string, 0, 1)
	r.values.Range(func(key, value any) bool {
		ring, _ := value.(*rings.Ring[*Registration])
		_, has := ring.Get(id)
		if has {
			ring.Remove(id)
			if ring.Len() == 0 {
				empties = append(empties, key.(string))
			}
		}
		return true
	})
	for _, empty := range empties {
		r.values.Delete(empty)
	}
	return
}

func (r *Registrations) GetExact(name string, id string) (registration *Registration, has bool) {
	v, loaded := r.values.Load(name)
	if !loaded || v == nil {
		return
	}
	ring, _ := v.(*rings.Ring[*Registration])
	registration, has = ring.Get(id)
	if !has || registration == nil {
		return
	}
	// todo check closed and error times
	return
}

func (r *Registrations) Get(name string, rvs RequestVersions) (registration *Registration, has bool) {
	v, loaded := r.values.Load(name)
	if !loaded || v == nil {
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
			continue
		}
		// todo check closed and error times
		if rvs == nil || len(rvs) == 0 {
			has = true
			return
		}
		if rvs.Accept(name, registration.version) {
			has = true
			return
		}
	}
	return
}

func (r *Registrations) Close() {
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
	return
}

func (r *Registrations) Ids() (ids []string) {
	ids = make([]string, 0, 1)
	r.values.Range(func(key, value any) bool {
		ring, _ := value.(*rings.Ring[*Registration])
		size := ring.Len()
		for i := 0; i < size; i++ {
			registration := ring.Next()
			if registration == nil {
				continue
			}
			id := registration.id
			ids = append(ids, id)
		}
		return true
	})
	sort.Strings(ids)
	return
}

func (r *Registrations) List() (values map[string]RegistrationList) {
	values = make(map[string]RegistrationList)
	r.values.Range(func(key, value any) bool {
		name := key.(string)
		group, has := values[name]
		if !has {
			group = make([]*Registration, 0, 1)
			values[name] = group
		}
		ring, _ := value.(*rings.Ring[*Registration])
		size := ring.Len()
		for i := 0; i < size; i++ {
			registration := ring.Next()
			if registration == nil {
				continue
			}
			group = append(group, registration)
		}
		return true
	})
	empties := make([]string, 0, 1)
	for name, list := range values {
		if len(list) == 0 {
			empties = append(empties, name)
			continue
		}
		sort.Sort(list)
	}
	for _, empty := range empties {
		delete(values, empty)
	}
	return
}

func (r *Registrations) AddNode(node Node) (err error) {
	address := strings.TrimSpace(node.Address)
	if address == "" {
		return
	}
	client, dialErr := r.dialer.Dial(address)
	if dialErr != nil {
		err = errors.Warning("fns: registrations dial node failed").WithCause(dialErr).
			WithMeta("address", address).
			WithMeta("nodeId", node.Id).WithMeta("node", node.Name)
		return
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	// todo 增加/services/names 判断是否为service

	// todo 增加捕获response header，如果存在Connection=close，则关闭中
	// todo http.StatusTooEarly

	header := http.Header{}
	header.Add(httpDeviceIdHeader, r.id)
	header.Add(httpRequestSignatureHeader, bytex.ToString(r.signer.Sign(bytex.FromString(r.id))))
	status, _, responseBody, getErr := client.Get(ctx, "/services/names?native=true", header)
	if getErr != nil {
		err = errors.Warning("fns: registrations get service names from node failed").
			WithCause(dialErr).
			WithMeta("address", address).
			WithMeta("nodeId", node.Id).WithMeta("node", node.Name)
		return
	}
	if status != http.StatusOK {
		if len(responseBody) == 0 {
			err = errors.Warning("fns: registrations get service names from node failed").
				WithMeta("address", address).
				WithMeta("status", strconv.Itoa(status)).
				WithMeta("nodeId", node.Id).WithMeta("node", node.Name)
			return
		}
		err = errors.Decode(responseBody)
		return
	}
	names := make([]string, 0, 1)
	decodeErr := json.Unmarshal(responseBody, &names)
	if decodeErr != nil {
		err = errors.Warning("fns: registrations get service names from node failed").
			WithMeta("address", address).WithCause(decodeErr).
			WithMeta("nodeId", node.Id).WithMeta("node", node.Name)
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

func (r *Registrations) MergeNodes(nodes Nodes) (err error) {
	existIds := r.Ids()
	if nodes == nil || len(nodes) == 0 {
		for _, id := range existIds {
			r.Remove(id)
		}
		return
	}
	sort.Sort(nodes)
	nodesLen := nodes.Len()
	existIdsLen := len(existIds)
	newNodes := make([]Node, 0, 1)
	diffNodeIds := make([]string, 0, 1)
	for _, node := range nodes {
		if existIdsLen != 0 && sort.SearchStrings(existIds, node.Id) < existIdsLen {
			continue
		}
		newNodes = append(newNodes, node)
	}
	if existIdsLen > 0 {
		for _, id := range existIds {
			exist := sort.Search(nodesLen, func(i int) bool {
				return nodes[i].Id == id
			}) < nodesLen
			if exist {
				continue
			}
			diffNodeIds = append(diffNodeIds, id)
		}
	}
	if len(diffNodeIds) > 0 {
		for _, id := range diffNodeIds {
			r.Remove(id)
		}
	}
	if len(newNodes) > 0 {
		for _, node := range newNodes {
			addErr := r.AddNode(node)
			if addErr != nil {
				err = errors.Warning("registrations: merge nodes failed").WithCause(addErr)
				return
			}
		}
	}
	return
}

func (r *Registrations) ListMembers(ctx context.Context) (members Nodes, err error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	nodes, getNodesErr := r.cluster.Nodes(ctx)
	if getNodesErr != nil {
		err = errors.Warning("registrations: list members failed").WithCause(getNodesErr)
		return
	}
	members = make([]Node, 0, 1)
	if nodes == nil || len(nodes) == 0 {
		return
	}
	for _, node := range nodes {
		if node.Id == r.id {
			continue
		}
		if node.Name == r.name && node.Version.Equals(r.version) {
			continue
		}
		members = append(members, node)
	}
	sort.Sort(members)
	return
}

func (r *Registrations) Refresh(ctx context.Context) {
	// todo
	timer := time.NewTimer(r.refreshInterval)
	for {
		stop := false
		select {
		case <-ctx.Done():
			stop = true
			break
		case <-timer.C:
			nodes, nodesErr := r.ListMembers(context.TODO())
			if nodesErr != nil {
				if r.log.WarnEnabled() {
					r.log.Warn().Cause(nodesErr).Message("registrations: refresh failed")
				}
				break
			}
			mergeErr := r.MergeNodes(nodes)
			if mergeErr != nil {
				if r.log.WarnEnabled() {
					r.log.Warn().Cause(mergeErr).Message("registrations: refresh failed")
				}
			}
			break
		}
		if stop {
			timer.Stop()
			break
		}
		timer.Reset(r.refreshInterval)
	}
	timer.Stop()
	return
}

func (r *Registrations) FetchDocuments() (v documents.Documents, err error) {
	// todo
	return
}

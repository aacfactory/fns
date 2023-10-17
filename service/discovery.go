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
	"github.com/aacfactory/fns/commons/window"
	"github.com/aacfactory/fns/service/documents"
	"github.com/aacfactory/fns/service/internal/secret"
	transports2 "github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/rings"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
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
	if task.r.Internal() {
		task.executeInternal(ctx)
		return
	}
	// non-internal is called by proxy
	// and its cache control was handled by middleware
	// also cache was update by remote endpoint
	// so just call
	registration := task.registration
	r := task.r
	fr := task.result

	// make request
	// request timeout
	timeout := task.timeout
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		timeout = deadline.Sub(time.Now())
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()
	// request body
	requestBody, encodeErr := json.Marshal(r.Argument())
	if encodeErr != nil {
		fr.Failed(errors.Warning("fns: registration request failed").WithCause(encodeErr))
		return
	}
	// request path
	header := r.Header()
	serviceName, fn := r.Fn()
	path := bytex.FromString(fmt.Sprintf("/%s/%s", serviceName, fn))
	// request
	req := transports2.NewUnsafeRequest(ctx, transports2.MethodPost, path)
	// request set header
	for name, vv := range header {
		for _, v := range vv {
			req.Header().Add(name, v)
		}
	}
	// request set body
	req.SetBody(requestBody)
	// do
	resp, postErr := registration.client.Do(ctx, req)
	if postErr != nil {
		task.registration.errs.Incr()
		fr.Failed(errors.Warning("fns: registration request failed").WithCause(postErr))
		return
	}
	// handle response
	// remote endpoint was closed
	if resp.Header.Get(httpConnectionHeader) == httpCloseHeader {
		task.registration.closed.Store(false)
		fr.Failed(ErrUnavailable)
		return
	}
	// failed response
	if resp.Status != http.StatusOK {
		var body errors.CodeError
		if resp.Body == nil || len(resp.Body) == 0 {
			body = errors.Warning("nil error")
		} else {
			body = errors.Decode(resp.Body)
		}
		task.registration.errs.Incr()
		fr.Failed(body)
		return
	}
	// succeed response
	if resp.Body == nil || len(resp.Body) == 0 {
		fr.Succeed(nil)
	} else {
		fr.Succeed(json.RawMessage(resp.Body))
	}
	return
}

func (task *registrationTask) executeInternal(ctx context.Context) {
	// internal is called by service function
	// so check cache first
	// when cache exists and was not out of date, then use cache
	// when cache not exist, then call with cache control disabled
	// when cache exist but was out of date, then call with if non match
	registration := task.registration
	r := task.r
	fr := task.result
	service, fn := r.Fn()
	trace, hasTracer := GetTracer(ctx)
	var span *Span
	if hasTracer {
		span = trace.StartSpan(service, fn)
		span.AddTag("kind", "remote")
	}

	ifNonMatch := ""
	var cachedBody []byte // is not internal response, cause cache was set in service, not in handler
	var cachedErr errors.CodeError
	if !r.Header().CacheControlDisabled() { // try cache control
		etag, status, deadline, body, exist := cacheControlFetch(ctx, r)
		if exist {
			// cache exists
			if status != http.StatusOK {
				cachedErr = errors.Decode(body)
			} else {
				cachedBody = body
			}
			// check deadline
			if deadline.After(time.Now()) {
				// not out of date
				if span != nil {
					span.Finish()
					span.AddTag("cached", "hit")
					span.AddTag("etag", etag)
					if cachedErr == nil {
						span.AddTag("status", "OK")
						span.AddTag("handled", "succeed")
					} else {
						span.AddTag("status", cachedErr.Name())
						span.AddTag("handled", "failed")
					}
				}
				if cachedErr == nil {
					fr.Succeed(body)
				} else {
					fr.Failed(cachedErr)
				}
				return
			} else {
				// out of date
				ifNonMatch = etag
			}
		}
	}
	// make request
	// request timeout
	timeout := task.timeout
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		timeout = deadline.Sub(time.Now())
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()
	// request body
	ir := internalRequest{
		User:     r.User(),
		Trunk:    r.Trunk(),
		Argument: r.Argument(),
	}
	requestBody, encodeErr := json.Marshal(&ir)
	if encodeErr != nil {
		// finish span
		err := errors.Warning("fns: registration request internal failed").WithCause(encodeErr)
		if span != nil {
			span.Finish()
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
		fr.Failed(err)
		return
	}
	// request path
	header := r.Header()
	path := bytex.FromString(fmt.Sprintf("/%s/%s", service, fn))
	// request
	req := transports2.NewUnsafeRequest(ctx, transports2.MethodPost, path)
	// request set header
	for name, vv := range header {
		for _, v := range vv {
			req.Header().Add(name, v)
		}
	}
	// internal sign header
	req.Header().Set(httpRequestInternalSignatureHeader, bytex.ToString(registration.signer.Sign(requestBody)))
	// if non match header
	if ifNonMatch != "" {
		req.Header().Set(httpCacheControlHeader, httpCacheControlEnabled)
		req.Header().Set(httpCacheControlIfNonMatch, ifNonMatch)
	}

	// request set body
	req.SetBody(requestBody)
	// do
	resp, postErr := registration.client.Do(ctx, req)
	if postErr != nil {
		task.registration.errs.Incr()
		// finish span
		err := errors.Warning("fns: registration request failed").WithCause(postErr)
		if span != nil {
			span.Finish()
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
		fr.Failed(err)
		return
	}
	// handle response
	if resp.Status != 200 && resp.Status != 304 && resp.Status >= 400 {
		// undefined error
		// remote endpoint was closed
		if resp.Header.Get(httpConnectionHeader) == httpCloseHeader {
			task.registration.closed.Store(false)
			if span != nil {
				span.Finish()
				span.AddTag("status", ErrUnavailable.Name())
				span.AddTag("handled", "failed")
			}
			fr.Failed(ErrUnavailable)
			return
		}
		if resp.Status == 404 {
			err := errors.NotFound("fns: not found").WithMeta("path", bytex.ToString(path))
			// finish span
			if span != nil {
				span.Finish()
				span.AddTag("status", err.Name())
				span.AddTag("handled", "failed")
			}
			fr.Failed(err)
			return
		}
		task.registration.errs.Incr()
		err := errors.Warning("fns: registration request failed").WithCause(errors.Warning(fmt.Sprintf("unknonw error, status is %d, %s", resp.Status, string(resp.Body))))
		// finish span
		if span != nil {
			span.Finish()
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
		fr.Failed(err)
		return
	}

	// check 304
	if resp.Status == http.StatusNotModified {
		// use cached
		if span != nil {
			span.Finish()
			span.AddTag("cached", "hit")
			span.AddTag("etag", ifNonMatch)
			if cachedErr == nil {
				span.AddTag("status", "OK")
				span.AddTag("handled", "succeed")
			} else {
				span.AddTag("status", cachedErr.Name())
				span.AddTag("handled", "failed")
			}
		}
		if cachedErr == nil {
			fr.Succeed(cachedBody)
		} else {
			fr.Failed(cachedErr)
		}
		return
	}
	// 200
	iresp := internalResponseImpl{}
	decodeErr := json.Unmarshal(resp.Body, &iresp)
	if decodeErr != nil {
		err := errors.Warning("fns: registration request failed").WithCause(errors.Warning("decode internal response failed")).WithCause(decodeErr)
		// finish span
		if span != nil {
			span.Finish()
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
		fr.Failed(err)
		return
	}
	var err errors.CodeError
	if !iresp.Succeed {
		err = errors.Decode(iresp.Body)
	} else {
		// user
		if iresp.User != nil && iresp.User.id != "" {
			r.User().SetId(iresp.User.id)
			if iresp.User.attributes != nil {
				r.User().SetAttributes(iresp.User.attributes)
			}
		}
		// trunk
		if iresp.Trunk != nil {
			iresp.Trunk.ForEach(func(key string, value []byte) (next bool) {
				r.Trunk().Put(key, value)
				next = true
				return
			})
		}
		// span
		if span != nil && iresp.Span != nil {
			span.AppendChild(iresp.Span)
		}
	}
	// finish span
	if span != nil {
		span.Finish()
		if err == nil {
			span.AddTag("status", "OK")
			span.AddTag("handled", "succeed")
		} else {
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
	}

	if err == nil {
		fr.Succeed(iresp.Body)
	} else {
		fr.Failed(err)
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
	devMode bool
	client  transports2.Client
	signer  *secret.Signer
	worker  Workers
	timeout time.Duration
	pool    sync.Pool
	closed  *atomic.Bool
	errs    *window.Times
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

func newRegistrations(log logs.Logger, id string, name string, version versions.Version, cluster Cluster, worker Workers, dialer transports2.Dialer, signer *secret.Signer, timeout time.Duration, refreshInterval time.Duration) *Registrations {
	dev, ok := cluster.(*devCluster)
	if ok {
		dialer = dev.dialer
	}
	return &Registrations{
		id:              id,
		name:            name,
		version:         version,
		log:             log,
		cluster:         cluster,
		devMode:         ok,
		values:          sync.Map{},
		nodes:           make(map[string]*Node),
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
	devMode         bool
	values          sync.Map
	nodes           map[string]*Node
	signer          *secret.Signer
	dialer          transports2.Dialer
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
	if registration.closed.Load() {
		r.Remove(registration.id)
		registration = nil
		has = false
		return
	}
	if registration.errs.Value() > 10 {
		registration = nil
		has = false
		return
	}
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
		if registration.closed.Load() {
			r.Remove(registration.id)
			continue
		}
		if registration.errs.Value() > 10 {
			continue
		}
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
			if registration.closed.Load() {
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

func (r *Registrations) FetchNodeDocuments(ctx context.Context, node *Node) (v documents.Documents, err error) {
	client, clientErr := r.dialer.Dial(node.Address)
	if clientErr != nil {
		err = errors.Warning("registrations: fetch node documents failed").WithCause(clientErr)
		return
	}

	req := transports2.NewUnsafeRequest(ctx, transports2.MethodGET, servicesDocumentsPath)
	req.Header().Set(httpDeviceIdHeader, r.id)

	for i := 0; i < 5; i++ {
		resp, doErr := client.Do(ctx, req)
		if doErr != nil {
			err = errors.Warning("registrations: fetch node documents failed").WithCause(doErr)
			return
		}
		if resp.Status == http.StatusTooEarly {
			time.Sleep(1 * time.Second)
			continue
		}
		if resp.Status != http.StatusOK {
			err = errors.Warning("registrations: fetch node documents failed")
			return
		}
		if resp.Body == nil || len(resp.Body) == 0 {
			return
		}
		v = documents.NewDocuments()
		decodeErr := json.Unmarshal(resp.Body, &v)
		if decodeErr != nil {
			err = errors.Warning("registrations: fetch node documents failed").WithCause(decodeErr)
			return
		}
		break
	}
	return
}

func (r *Registrations) AddNode(node *Node) (err error) {
	if node.Services == nil || len(node.Services) == 0 {
		return
	}
	client, clientErr := r.dialer.Dial(node.Address)
	if clientErr != nil {
		err = errors.Warning("registrations: add node failed").WithCause(clientErr)
		return
	}

	for _, svc := range node.Services {
		registration := &Registration{
			hostId:  r.id,
			id:      node.Id,
			version: node.Version,
			address: node.Address,
			devMode: r.devMode,
			name:    svc.Name,
			client:  client,
			signer:  r.signer,
			worker:  r.worker,
			timeout: r.timeout,
			pool:    sync.Pool{},
			closed:  &atomic.Bool{},
			errs:    window.NewTimes(10 * time.Second),
		}
		registration.pool.New = func() any {
			return newRegistrationTask(registration, registration.timeout, func(task *registrationTask) {
				registration.release(task)
			})
		}
		r.Add(registration)
	}
	r.nodes[node.Id] = node
	return
}

func (r *Registrations) MergeNodes(nodes Nodes) (err error) {
	existNodes := r.nodes
	if nodes == nil || len(nodes) == 0 {
		for _, existNode := range existNodes {
			r.Remove(existNode.Id)
		}
		r.nodes = make(map[string]*Node)
		return
	}
	sort.Sort(nodes)
	nodesLen := nodes.Len()
	newNodes := make([]*Node, 0, 1)
	diffNodes := make([]*Node, 0, 1)
	for _, node := range nodes {
		_, exist := r.nodes[node.Id]
		if exist {
			continue
		}
		newNodes = append(newNodes, node)
	}
	for _, existNode := range r.nodes {
		exist := sort.Search(nodesLen, func(i int) bool {
			return nodes[i].Id == existNode.Id
		}) < nodesLen
		if exist {
			continue
		}
		diffNodes = append(diffNodes)
	}

	if len(diffNodes) > 0 {
		for _, node := range diffNodes {
			r.Remove(node.Id)
			delete(r.nodes, node.Id)
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
	nodes, getNodesErr := r.cluster.Nodes(ctx)
	if getNodesErr != nil {
		err = errors.Warning("registrations: list members failed").WithCause(getNodesErr)
		return
	}
	members = make([]*Node, 0, 1)
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
		if node.Services != nil && len(node.Services) > 0 {
			members = append(members, node)
			continue
		}
		if node.Services == nil || len(node.Services) == 0 {
			continue
		}
		members = append(members, node)
	}
	sort.Sort(members)
	return
}

func (r *Registrations) Refresh(ctx context.Context) {
	timer := time.NewTimer(10 * time.Millisecond)
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

func (r *Registrations) ServiceDocuments() (v documents.Documents, err error) {
	v = documents.NewDocuments()
	if r.nodes == nil || len(r.nodes) == 0 {
		return
	}
	for _, node := range r.nodes {
		if len(node.Services) == 0 {
			continue
		}
		docs := documents.NewDocuments()
		for _, service := range node.Services {
			doc := service.Document
			if doc == nil {
				continue
			}
			docs.Add(doc)
		}
		v = v.Merge(docs)
	}
	return
}

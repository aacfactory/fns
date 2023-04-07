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
	"github.com/aacfactory/fns/commons/caches"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/documents"
	"github.com/aacfactory/fns/service/internal/commons/window"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/rings"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"net/http"
	"sort"
	"strconv"
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

func (task *registrationTask) getCached(r Request) (v []byte, has bool) {
	buf := bytebufferpool.Get()
	// deviceId
	deviceId := r.Header().DeviceId()
	_, _ = buf.Write(bytex.FromString(deviceId))
	// path
	service, fn := r.Fn()
	_, _ = buf.Write(bytex.FromString(fmt.Sprintf("/%s/%s", service, fn)))
	// body
	body, bodyErr := json.Marshal(r.Argument())
	if bodyErr != nil {
		bytebufferpool.Put(buf)
		return
	}
	_, _ = buf.Write(body)
	k := bytex.FromString(strconv.FormatUint(xxhash.Sum64(buf.Bytes()), 10))
	bytebufferpool.Put(buf)
	v, has = task.registration.cache.Get(k)
	return
}

func (task *registrationTask) setCached(k []byte, v []byte, ttl time.Duration) {
	// no error cause k is etag and v is json
	_ = task.registration.cache.SetWithTTL(k, v, ttl)
	return
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

	cachedResult, cached := task.getCached(r)

	if r.Internal() {
		if cached {
			ir := &internalResponseImpl{}
			decodeErr := json.Unmarshal(cachedResult, ir)
			if decodeErr != nil {
				fr.Failed(errors.Warning("fn: registration request internal failed").WithCause(decodeErr))
				return
			}
			var resultErr errors.CodeError
			if !ir.Succeed {
				resultErr = errors.Decode(ir.Body)
			}
			// span
			trace, hasTracer := GetTracer(ctx)
			if hasTracer {
				service, fn := r.Fn()
				span := &Span{
					Id_:         uid.UID(),
					Service_:    service,
					Fn_:         fn,
					TracerId_:   trace.Id(),
					StartAT_:    time.Now(),
					FinishedAT_: time.Now(),
					Children_:   make([]*Span, 0, 1),
					Tags_:       make(map[string]string),
					parent:      nil,
				}
				if ir.Succeed {
					span.AddTag("status", "OK")
					span.AddTag("handled", "succeed")
					span.AddTag("cached", "true")
				} else {
					span.AddTag("status", resultErr.Name())
					span.AddTag("handled", "failed")
					span.AddTag("cached", "true")
				}
				trace.Span().AppendChild(span)
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
				fr.Failed(resultErr)
			}
			return
		}
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
		header.Set(httpRequestInternalHeader, "true")
		header.Set(httpRequestIdHeader, r.Id())
		header.Set(httpRequestInternalSignatureHeader, bytex.ToString(registration.signer.Sign(requestBody)))
		header.Set(httpRequestTimeoutHeader, fmt.Sprintf("%d", uint64(timeout/time.Millisecond)))
		if task.registration.devMode {
			header.Set(httpDevModeHeader, task.registration.id)
		}

		serviceName, fn := r.Fn()
		req := transports.NewUnsafeRequest(ctx, transports.MethodPost, bytex.FromString(fmt.Sprintf("/%s/%s", serviceName, fn)))

		for name, vv := range header {
			for _, v := range vv {
				req.Header().Add(name, v)
			}
		}

		req.SetBody(requestBody)
		resp, postErr := registration.client.Do(ctx, req)
		if postErr != nil {
			task.registration.errs.Incr()
			fr.Failed(errors.Warning("fns: registration request internal failed").WithCause(postErr))
			return
		}

		if resp.Header.Get(httpConnectionHeader) == httpCloseHeader {
			task.registration.closed.Store(false)
			fr.Failed(ErrUnavailable)
			return
		}

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

		ir := &internalResponseImpl{}
		decodeErr := json.Unmarshal(resp.Body, ir)
		if decodeErr != nil {
			fr.Failed(errors.Warning("fns: registration request internal failed").WithCause(decodeErr))
			return
		}
		// span
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
		// cache
		etag := resp.Header.Get(httpETagHeader)
		if etag != "" {
			etagTTL := time.Duration(0)
			if ttl := resp.Header.Get(httpResponseCacheTTL); ttl != "" {
				etagTTL, _ = time.ParseDuration(ttl)
			}
			if etagTTL < 1 {
				etagTTL = registration.cacheDefaultTTL
			}
			task.setCached(bytex.FromString(etag), resp.Body, etagTTL)
		}
	} else {
		if cached {
			cachedResultDataLen := len(cachedResult) - 1
			if cachedResult[cachedResultDataLen] == '1' {
				if cachedResultDataLen == 0 {
					fr.Succeed(nil)
				} else {
					fr.Succeed(cachedResult[0:cachedResultDataLen])
				}
			} else {
				fr.Failed(errors.Decode(cachedResult[0:cachedResultDataLen]))
			}
			return
		}

		requestBody, encodeErr := json.Marshal(r.Argument())
		if encodeErr != nil {
			fr.Failed(errors.Warning("fns: registration request failed").WithCause(encodeErr))
			return
		}
		header := r.Header()
		serviceName, fn := r.Fn()

		req := transports.NewUnsafeRequest(ctx, transports.MethodPost, bytex.FromString(fmt.Sprintf("/%s/%s", serviceName, fn)))

		for name, vv := range header {
			for _, v := range vv {
				req.Header().Add(name, v)
			}
		}
		req.SetBody(requestBody)
		resp, postErr := registration.client.Do(ctx, req)
		if postErr != nil {
			task.registration.errs.Incr()
			fr.Failed(errors.Warning("fns: registration request failed").WithCause(postErr))
			return
		}

		if resp.Header.Get(httpConnectionHeader) == httpCloseHeader {
			task.registration.closed.Store(false)
			fr.Failed(ErrUnavailable)
			return
		}
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
		if resp.Body == nil || len(resp.Body) == 0 {
			fr.Succeed(nil)
		} else {
			fr.Succeed(json.RawMessage(resp.Body))
		}
		// cache
		etag := resp.Header.Get(httpETagHeader)
		if etag != "" {
			etagTTL := time.Duration(0)
			if ttl := resp.Header.Get(httpResponseCacheTTL); ttl != "" {
				etagTTL, _ = time.ParseDuration(ttl)
			}
			if etagTTL < 1 {
				etagTTL = registration.cacheDefaultTTL
			}
			body := resp.Body
			if resp.Body == nil {
				body = make([]byte, 0, 1)
			}
			task.setCached(bytex.FromString(etag), body, etagTTL)
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
	hostId          string
	id              string
	version         versions.Version
	address         string
	name            string
	devMode         bool
	client          transports.Client
	signer          *secret.Signer
	worker          Workers
	timeout         time.Duration
	pool            sync.Pool
	closed          *atomic.Bool
	errs            *window.Times
	cache           *caches.Cache
	cacheDefaultTTL time.Duration
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

func newRegistrations(log logs.Logger, id string, name string, version versions.Version, cluster Cluster, worker Workers, dialer transports.Dialer, signer *secret.Signer, timeout time.Duration, refreshInterval time.Duration, clusterCache *caches.Cache, cacheDefaultTTL time.Duration) *Registrations {
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
		cache:           clusterCache,
		cacheDefaultTTL: cacheDefaultTTL,
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
	dialer          transports.Dialer
	worker          Workers
	timeout         time.Duration
	refreshInterval time.Duration
	cache           *caches.Cache
	cacheDefaultTTL time.Duration
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

	req := transports.NewUnsafeRequest(ctx, transports.MethodGET, bytex.FromString("/services/documents"))
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

	for _, name := range node.Services {
		registration := &Registration{
			hostId:          r.id,
			id:              node.Id,
			version:         node.Version,
			address:         node.Address,
			devMode:         r.devMode,
			name:            name,
			client:          client,
			signer:          r.signer,
			worker:          r.worker,
			timeout:         r.timeout,
			pool:            sync.Pool{},
			closed:          &atomic.Bool{},
			errs:            window.NewTimes(10 * time.Second),
			cache:           r.cache,
			cacheDefaultTTL: r.cacheDefaultTTL,
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
		names, nameErr := r.GetNodeServices(ctx, node)
		if nameErr != nil {
			continue
		}
		if names == nil || len(names) == 0 {
			continue
		}
		node.Services = names
		members = append(members, node)
	}
	sort.Sort(members)
	return
}

func (r *Registrations) GetNodeServices(ctx context.Context, node *Node) (names []string, err error) {
	client, clientErr := r.dialer.Dial(node.Address)
	if clientErr != nil {
		err = clientErr
		return
	}
	req := transports.NewUnsafeRequest(ctx, transports.MethodGET, bytex.FromString("/services/names"))
	req.Header().Set(httpDeviceIdHeader, r.id)
	req.Header().Set(httpRequestInternalSignatureHeader, bytex.ToString(r.signer.Sign(bytex.FromString(r.id))))

	resp, doErr := client.Do(ctx, req)
	if doErr != nil {
		err = doErr
		return
	}
	if resp.Status != http.StatusOK {
		return
	}
	names = make([]string, 0, 1)
	_ = json.Unmarshal(resp.Body, names)
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

func (r *Registrations) FetchDocuments(ctx context.Context) (v documents.Documents, err error) {
	v = documents.NewDocuments()
	if r.nodes == nil || len(r.nodes) == 0 {
		return
	}
	for _, node := range r.nodes {
		doc, docErr := r.FetchNodeDocuments(ctx, node)
		if docErr != nil {
			continue
		}
		if doc == nil || doc.Len() == 0 {
			continue
		}
		v = v.Merge(doc)
	}
	return
}

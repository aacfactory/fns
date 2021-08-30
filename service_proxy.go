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
	"github.com/aacfactory/json"
	"github.com/valyala/fasthttp"
	"sync"
	"sync/atomic"
	"time"
)

func NewLocaledServiceProxy(service Service) *LocaledServiceProxy {
	return &LocaledServiceProxy{
		id:      UID(),
		service: service,
	}
}

type LocaledServiceProxy struct {
	id      string
	service Service
}

func (proxy *LocaledServiceProxy) Id() string {
	return proxy.id
}

func (proxy *LocaledServiceProxy) Request(ctx Context, fn string, argument Argument) (result Result) {
	result = SyncResult()
	v, err := proxy.service.Handle(WithNamespace(ctx, proxy.service.Namespace()), fn, argument)
	if err != nil {
		result.Failed(err)
		return
	}
	result.Succeed(v)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewRemotedServiceProxyGroup(namespace string) (group *RemotedServiceProxyGroup) {
	group = &RemotedServiceProxyGroup{
		mutex:     sync.RWMutex{},
		namespace: namespace,
		pos:       0,
		keys:      make([]string, 0, 1),
		agentMap:  make(map[string]*RemotedServiceProxy),
	}
	return
}

type RemotedServiceProxyGroup struct {
	mutex     sync.RWMutex
	namespace string
	pos       uint64
	keys      []string
	agentMap  map[string]*RemotedServiceProxy
}

func (group *RemotedServiceProxyGroup) Namespace() string {
	return group.namespace
}

func (group *RemotedServiceProxyGroup) Next() (proxy *RemotedServiceProxy, err errors.CodeError) {
	group.mutex.RLock()
	defer group.mutex.RUnlock()

	num := uint64(len(group.keys))
	if num < 1 {
		err = errors.New(555, "***WARNING***", "fns GroupRemotedServiceProxy Next: no agents")
		return
	}

	pos := atomic.LoadUint64(&group.pos) % num
	atomic.AddUint64(&group.pos, 1)

	key := group.keys[pos]
	agent, has := group.agentMap[key]

	if !has {
		err = errors.New(555, "***WARNING***", "fns GroupRemotedServiceProxy Next: no agents")
		return
	}

	if !agent.Health() {

		group.mutex.RUnlock()
		group.RemoveAgent(agent.id)
		group.mutex.RLock()

		group.mutex.RUnlock()
		proxy, err = group.Next()
		group.mutex.RLock()

		return
	}

	proxy = agent

	return
}

func (group *RemotedServiceProxyGroup) GetAgent(id string) (proxy *RemotedServiceProxy, err errors.CodeError) {
	group.mutex.RLock()
	defer group.mutex.RUnlock()
	agent, has := group.agentMap[id]
	if !has {
		err = errors.New(555, "***WARNING***", "fns GroupRemotedServiceProxy GetAgent: no such agent")
		return
	}
	proxy = agent
	return
}

func (group *RemotedServiceProxyGroup) AppendAgent(agent *RemotedServiceProxy) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	if agent.namespace != group.namespace {
		return
	}

	if !agent.Check() {
		return
	}

	id := agent.Id()

	_, has := group.agentMap[id]
	if has {
		return
	}

	group.keys = append(group.keys, id)
	group.agentMap[id] = agent

	return
}

func (group *RemotedServiceProxyGroup) ContainsAgent(id string) (has bool) {
	group.mutex.RLock()
	defer group.mutex.RUnlock()
	_, has = group.agentMap[id]
	return
}

func (group *RemotedServiceProxyGroup) AgentNum() (num int) {
	group.mutex.RLock()
	defer group.mutex.RUnlock()
	num = len(group.agentMap)
	return
}

func (group *RemotedServiceProxyGroup) RemoveAgent(id string) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	agent, has := group.agentMap[id]
	if !has {
		return
	}

	agent.Close()

	newKeys := make([]string, 0, 1)
	for _, key := range group.keys {
		if key == id {
			continue
		}
		newKeys = append(newKeys, key)
	}
	group.keys = newKeys

	delete(group.agentMap, id)

}

func (group *RemotedServiceProxyGroup) Close() {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	for _, agent := range group.agentMap {
		agent.Close()
	}

	group.keys = make([]string, 0, 1)
	group.agentMap = make(map[string]*RemotedServiceProxy)

	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewRemotedServiceProxy(id string, namespace string, address string) (proxy *RemotedServiceProxy) {

	proxy = &RemotedServiceProxy{
		id:        id,
		namespace: namespace,
		address:   address,
		client: &fasthttp.Client{
			Name: id,
		},
	}

	proxy.Check()

	return
}

var (
	get  = []byte("GET")
	post = []byte("POST")
)

type RemotedServiceProxy struct {
	id        string
	namespace string
	address   string
	client    *fasthttp.Client
	health    int64
}

func (proxy *RemotedServiceProxy) Namespace() string {
	return proxy.namespace
}

func (proxy *RemotedServiceProxy) Request(ctx Context, fn string, argument Argument) (result Result) {
	result = SyncResult()

	if !proxy.Health() {
		result.Failed(errors.New(555, "***WARNING***", fmt.Sprintf("fns Proxy Request: [%s:%s:%s] is not healthy", proxy.namespace, proxy.address, proxy.id)))
		return
	}

	body, bodyErr := argument.MarshalJSON()
	if bodyErr != nil {
		result.Failed(errors.New(555, "***WARNING***", fmt.Sprintf("fns Proxy Request: encode argument failed")).WithCause(bodyErr))
		return
	}

	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	request.URI().SetHost(proxy.address)
	request.URI().SetPath(fmt.Sprintf("/%s/%s", proxy.namespace, fn))
	request.Header.SetMethodBytes(post)
	request.Header.SetContentTypeBytes(jsonUTF8ContentType)
	request.Header.SetBytesK(requestIdHeader, ctx.RequestId())
	request.Header.SetBytesKV(requestMetaHeader, ctx.Meta().Encode())
	request.Header.SetBytesKV(authorizationHeader, ctx.Authorization())

	request.SetBody(body)

	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(response)

	doErr := proxy.client.DoTimeout(request, response, 5*time.Second)
	if doErr != nil {
		if doErr == fasthttp.ErrTimeout {
			result.Failed(errors.Timeout(fmt.Sprintf("fns Proxy Request: post to %s timeout", proxy.address)).WithCause(doErr))
			proxy.Check()
		} else {
			result.Failed(errors.New(555, "***WARNING***", fmt.Sprintf("fns Proxy Request: post to %s failed", proxy.address)).WithCause(doErr))
			proxy.Check()
		}
		return
	}

	if response.StatusCode() == 200 {
		result.Succeed(response.Body())
	} else {
		err := errors.ServiceError("")
		decodeErr := json.Unmarshal(response.Body(), err)
		if decodeErr != nil {
			result.Failed(errors.New(555, "***WARNING***", fmt.Sprintf("fns Proxy Request: decode body %s failed", string(response.Body()))).WithCause(decodeErr))
			return
		}
		result.Failed(err)
	}

	return
}

func (proxy *RemotedServiceProxy) Id() string {
	return proxy.id
}

func (proxy *RemotedServiceProxy) Health() (ok bool) {
	return atomic.LoadInt64(&proxy.health) == int64(1)
}

func (proxy *RemotedServiceProxy) Check() (ok bool) {

	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	request.URI().SetHost(proxy.address)
	request.URI().SetPath(healthCheckPath)
	request.Header.SetMethodBytes(get)

	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(response)

	err := proxy.client.DoTimeout(request, response, 5*time.Second)
	if err != nil {
		return
	}

	ok = response.StatusCode() == 200

	if ok {
		atomic.StoreInt64(&proxy.health, int64(1))
	} else {
		atomic.StoreInt64(&proxy.health, int64(0))
	}

	return
}

func (proxy *RemotedServiceProxy) Close() {
	proxy.client.CloseIdleConnections()
	return
}

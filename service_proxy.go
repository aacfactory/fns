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

type LocaledServiceProxy struct {
	Service Service
}

func (proxy *LocaledServiceProxy) Request(ctx Context, fn string, argument Argument) (result Result) {
	result = SyncResult()
	v, err := proxy.Service.Handle(WithNamespace(ctx, proxy.Service.Namespace()), fn, argument)
	if err != nil {
		result.Failed(err)
		return
	}
	result.Succeed(v)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewGroupRemotedServiceProxy(namespace string) (group *GroupRemotedServiceProxy) {
	group = &GroupRemotedServiceProxy{
		mutex:     sync.Mutex{},
		wg:        sync.WaitGroup{},
		namespace: namespace,
		pos:       0,
		keys:      make([]string, 0, 1),
		agentMap:  make(map[string]*RemotedServiceProxy),
	}
	return
}

type GroupRemotedServiceProxy struct {
	mutex     sync.Mutex
	wg        sync.WaitGroup
	namespace string
	pos       uint64
	keys      []string
	agentMap  map[string]*RemotedServiceProxy
}

func (proxy *GroupRemotedServiceProxy) Request(ctx Context, fn string, argument Argument) (result Result) {
	proxy.wg.Wait()

	pos := proxy.pos & uint64(len(proxy.keys))
	key := proxy.keys[pos]
	agent, has := proxy.agentMap[key]

	atomic.AddUint64(&proxy.pos, 1)

	if !has {
		result = SyncResult()
		result.Failed(errors.New(555, "***WARNING***", "fns GroupRemotedServiceProxy Request: next agent failed"))
		return
	}

	if !agent.Health() {
		proxy.RemoveAgent(agent.id)

		result = SyncResult()
		result.Failed(errors.New(555, "***WARNING***", fmt.Sprintf("fns Proxy Request: [%s:%s:%s] is not healthy", agent.namespace, agent.address, agent.id)))
		return
	}

	result = agent.Request(ctx, fn, argument)

	return
}

func (proxy *GroupRemotedServiceProxy) AppendAgent(agent *RemotedServiceProxy) {
	proxy.wg.Add(1)
	defer proxy.wg.Done()

	proxy.mutex.Lock()
	defer proxy.mutex.Unlock()

	if agent.namespace != proxy.namespace {
		return
	}

	id := agent.Id()

	_, has := proxy.agentMap[id]
	if has {
		return
	}

	proxy.keys = append(proxy.keys, id)
	proxy.agentMap[id] = agent

	return
}

func (proxy *GroupRemotedServiceProxy) ContainsAgent(id string) (has bool) {
	_, has = proxy.agentMap[id]
	return
}

func (proxy *GroupRemotedServiceProxy) AgentNum() (num int) {
	num = len(proxy.agentMap)
	return
}

func (proxy *GroupRemotedServiceProxy) RemoveAgent(id string) {
	proxy.wg.Add(1)
	defer proxy.wg.Done()

	proxy.mutex.Lock()
	defer proxy.mutex.Unlock()

	agent, has := proxy.agentMap[id]
	if !has {
		return
	}

	agent.Close()

	newKeys := make([]string, 0, 1)
	for _, key := range proxy.keys {
		if key == id {
			continue
		}
		newKeys = append(newKeys, key)
	}
	proxy.keys = newKeys

	delete(proxy.agentMap, id)

}

func (proxy *GroupRemotedServiceProxy) Close() {
	proxy.wg.Add(1)
	defer proxy.wg.Done()

	proxy.mutex.Lock()
	defer proxy.mutex.Unlock()

	for _, agent := range proxy.agentMap {
		agent.Close()
	}

	proxy.keys = make([]string, 0, 1)
	proxy.agentMap = make(map[string]*RemotedServiceProxy)

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

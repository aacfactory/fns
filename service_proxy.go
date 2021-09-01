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
	"runtime"
	"sync/atomic"
	"time"
)

func NewLocaledServiceProxy(service Service) *LocaledServiceProxy {
	return &LocaledServiceProxy{
		service: service,
	}
}

type LocaledServiceProxy struct {
	service Service
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

func NewHttpClients(poolSize int) (hc *HttpClients) {
	if poolSize < 1 {
		poolSize = runtime.NumCPU() * 2
	}
	clients := make([]*fasthttp.Client, 0, 1)
	for i := 0; i < poolSize; i++ {
		clients = append(clients, &fasthttp.Client{
			Name: "FNS",
		})
	}
	idx := uint64(0)
	hc = &HttpClients{
		idx:     &idx,
		size:    uint64(poolSize),
		clients: clients,
	}
	return
}

type HttpClients struct {
	idx     *uint64
	size    uint64
	clients []*fasthttp.Client
}

func (hc *HttpClients) next() (client *fasthttp.Client) {
	client = hc.clients[*hc.idx%hc.size]
	atomic.AddUint64(hc.idx, 1)
	return
}

func (hc *HttpClients) Check(registration Registration) (ok bool) {
	client := hc.next()
	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	request.URI().SetHost(registration.Address)
	request.URI().SetPath(healthCheckPath)
	request.Header.SetMethodBytes(get)

	response := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(response)

	err := client.DoTimeout(request, response, 2*time.Second)
	if err != nil {
		return
	}

	ok = response.StatusCode() == 200
	return
}

func (hc *HttpClients) Close() {
	for _, client := range hc.clients {
		client.CloseIdleConnections()
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewRemotedServiceProxy(clients *HttpClients, registration *Registration, problemCh chan<- *Registration) (proxy *RemotedServiceProxy) {
	proxy = &RemotedServiceProxy{
		registration: registration,
		client:       clients.next(),
		problemCh:    problemCh,
	}
	return
}

var (
	get  = []byte("GET")
	post = []byte("POST")
)

type RemotedServiceProxy struct {
	registration *Registration
	client       *fasthttp.Client
	problemCh    chan<- *Registration
}

func (proxy *RemotedServiceProxy) Request(ctx Context, fn string, argument Argument) (result Result) {
	result = SyncResult()

	body, bodyErr := argument.MarshalJSON()
	if bodyErr != nil {
		result.Failed(errors.New(555, "***WARNING***", fmt.Sprintf("fns Proxy Request: encode argument failed")).WithCause(bodyErr))
		return
	}

	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	request.URI().SetHost(proxy.registration.Address)
	request.URI().SetPath(fmt.Sprintf("/%s/%s", proxy.registration.Namespace, fn))
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
			result.Failed(errors.Timeout(fmt.Sprintf("fns Proxy Request: post to %s timeout", proxy.registration.Address)).WithCause(doErr))
		} else {
			result.Failed(errors.New(555, "***WARNING***", fmt.Sprintf("fns Proxy Request: post to %s failed", proxy.registration.Address)).WithCause(doErr))
		}
		if proxy.registration.Id != "" {
			proxy.problemCh <- proxy.registration
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

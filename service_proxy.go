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
	"encoding/binary"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/secret"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
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
	v, err := proxy.service.handle(WithNamespace(ctx, proxy.service.namespace()), fn, argument)
	if err != nil {
		result.Failed(err)
		return
	}
	if v == nil {
		v = &Empty{}
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
	get                 = []byte("GET")
	post                = []byte("POST")
	fnsProxyContentType = []byte("application/fns+proxy")
)

type RemotedServiceProxy struct {
	registration *Registration
	client       *fasthttp.Client
	problemCh    chan<- *Registration
}

func (proxy *RemotedServiceProxy) Request(ctx Context, fn string, argument Argument) (result Result) {
	result = SyncResult()
	body, encodeBodyErr := proxyMessageEncode(ctx.Meta(), argument)
	if encodeBodyErr != nil {
		result.Failed(errors.New(555, "***WARNING***", fmt.Sprintf("fns Proxy Request: encode body failed")).WithCause(encodeBodyErr))
		return
	}
	requestId := ctx.RequestId()
	buf := bytebufferpool.Get()
	_, _ = buf.WriteString(requestId)
	_, _ = buf.Write(body)
	signedTarget := buf.Bytes()
	bytebufferpool.Put(buf)
	signature := secret.Sign(signedTarget, secretKey)
	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	request.URI().SetHost(proxy.registration.Address)
	request.URI().SetPath(fmt.Sprintf("/%s/%s", proxy.registration.Namespace, fn))
	request.Header.SetMethodBytes(post)
	request.Header.SetContentTypeBytes(fnsProxyContentType)
	request.Header.SetBytesK(requestIdHeader, ctx.RequestId())
	request.Header.SetBytesKV(requestSignHeader, signature)
	authorization, hasAuthorization := ctx.User().Authorization()
	if hasAuthorization {
		request.Header.SetBytesKV(authorizationHeader, authorization)
	}
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
		if response.Body() == nil || len(response.Body()) == 0 {
			result.Succeed(nil)
		} else {
			result.Succeed(response.Body())
		}
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

// +-------------------------------------------------------------------------------------------------------------------+

func proxyMessageEncode(meta ContextMeta, argument Argument) (p []byte, err errors.CodeError) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	metaValue := meta.Encode()
	argumentValue, encodeArgErr := argument.MarshalJSON()
	if encodeArgErr != nil {
		err = errors.Warning("fns Proxy: encode message failed").WithCause(encodeArgErr)
		return
	}
	lewErr := binary.Write(buf, binary.LittleEndian, uint32(len(metaValue)))
	if lewErr != nil {
		err = errors.Warning("fns Proxy: encode message failed").WithCause(lewErr)
		return
	}
	_, _ = buf.Write(metaValue)
	_, _ = buf.Write(argumentValue)
	p = buf.Bytes()
	return
}

func proxyMessageDecode(p []byte) (meta []byte, argument []byte) {
	n := binary.LittleEndian.Uint32(p[0:4])
	meta = p[4 : 4+n]
	argument = p[4+n:]
	return
}

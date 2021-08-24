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
	"encoding/json"
	"fmt"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"net"
	"strings"
	"sync"
	"time"
)


// +-------------------------------------------------------------------------------------------------------------------+

func discoveryRegistrationMapToHttpClient(registrations []Registration) (client FnHttpClient, err error) {
	lb := &fasthttp.LBClient{
		Clients: make([]fasthttp.BalancingClient, 0, len(registrations)),
		Timeout: 30 * time.Second,
		HealthCheck: func(req *fasthttp.Request, resp *fasthttp.Response, err error) (ok bool) {
			if err != nil && err == fasthttp.ErrTimeout {
				ok = false
				return
			}
			ok = true
			return
		},
	}
	for _, registration := range registrations {
		hostClient := &fasthttp.HostClient{
			Addr: registration.Address,
		}
		if registration.ClientTLS.Enable {
			hostClient.IsTLS = true
			tlsConfig, tlcConfigErr := registration.ClientTLS.Config()
			if tlcConfigErr != nil {
				err = fmt.Errorf("make client tls config failed, namespace is %s, address is %s", registration.Name, registration.Address)
				return
			}
			hostClient.TLSConfig = tlsConfig
		}
		lb.Clients = append(lb.Clients, hostClient)
	}
	client = newFastHttpLBFnHttpClient(lb)
	return
}

type fnRemoteBody struct {
	Meta ContextMeta     `json:"meta,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type FnHttpClient interface {
	Request(ctx Context, arg Argument) (response []byte, err error)
}

func newFastHttpLBFnHttpClient(lb *fasthttp.LBClient) FnHttpClient {
	return &fastHttpLBFnHttpClient{
		lb: lb,
	}
}

type fastHttpLBFnHttpClient struct {
	lb *fasthttp.LBClient
}

func (c *fastHttpLBFnHttpClient) Request(ctx Context, arg Argument) (data []byte, err error) {
	namespace := ctx.Meta().Namespace()
	fnName := ctx.Meta().FnName()

	// request
	request := &fasthttp.Request{}
	request.SetRequestURI("/fc")
	request.Header.Set("Fns-Namespace", namespace)
	request.Header.Set("Fns-Name", fnName)
	request.Header.Set("Fns-Request-Id", ctx.Meta().RequestId())
	bodyMap := fnRemoteBody{
		Meta: ctx.Meta(),
		Data: arg.Data(),
	}
	body, bodyEncodeErr := JsonAPI().Marshal(bodyMap)
	if bodyEncodeErr != nil {
		err = ServiceError(fmt.Sprintf("call remote %s/%s failed, %v", namespace, fnName, bodyEncodeErr))
		return
	}
	request.SetBody(body)

	// response
	response := &fasthttp.Response{}

	remoteErr := c.lb.Do(request, response)
	if remoteErr != nil {
		err = ServiceError(fmt.Sprintf("call remote %s/%s failed, %v", namespace, fnName, remoteErr))
		return
	}
	contentEncoding := response.Header.Peek("Content-Encoding")
	if contentEncoding == nil || len(contentEncoding) == 0 {
		data = response.Body()
	} else if string(contentEncoding) == "gzip" {
		data, err = response.BodyGunzip()
		if err != nil {
			err = ServiceError(fmt.Sprintf("call remote %s/%s failed, %v", namespace, fnName, err))
			return
		}
	} else if string(contentEncoding) == "deflate" {
		data, err = response.BodyInflate()
		if err != nil {
			err = ServiceError(fmt.Sprintf("call remote %s/%s failed, %v", namespace, fnName, err))
			return
		}
	} else if string(contentEncoding) == "br" {
		data, err = response.BodyUnbrotli()
		if err != nil {
			err = ServiceError(fmt.Sprintf("call remote %s/%s failed, %v", namespace, fnName, err))
			return
		}
	} else {
		err = ServiceError(fmt.Sprintf("call remote %s/%s failed, content encoding %s is not supported", namespace, fnName, string(contentEncoding)))
		return
	}
	buf := bytebufferpool.Get()
	if response.StatusCode() == 200 {
		_ = buf.WriteByte('1')
	} else {
		_ = buf.WriteByte('0')
	}
	_, _ = buf.Write(data)
	data = buf.Bytes()
	bytebufferpool.Put(buf)
	return
}

func NewWhiteList(cidr []string) (w *WhiteList, err error) {
	if cidr == nil || len(cidr) == 0 {
		err = fmt.Errorf("new white list failed, cidr is nil")
		return
	}
	w = &WhiteList{Ips: make([]*net.IPNet, 0, 1), wg: &sync.WaitGroup{}}
	wErr := w.ReWriteIpList(cidr)
	if wErr != nil {
		err = wErr
		w = nil
		return
	}
	return
}

type WhiteList struct {
	Ips []*net.IPNet
	wg  *sync.WaitGroup
}

func (w *WhiteList) ReWriteIpList(netIps []string) (err error) {
	w.wg.Add(1)
	defer w.wg.Done()
	for _, netIp0 := range netIps {
		netIp0 = strings.TrimSpace(netIp0)
		if netIp0 == "" {
			err = fmt.Errorf("net ip is bad")
			return
		}

		_, ipn, parseErr := net.ParseCIDR(netIp0)
		if parseErr != nil {
			err = fmt.Errorf("make white list failed, bad net ip, %s", netIp0)
			return
		}
		w.Ips = append(w.Ips, ipn)
	}
	return
}

func (w *WhiteList) Contains(requestIp0 string) (succeed bool, err error) {
	w.wg.Wait()
	if requestIp0 == "" {
		err = fmt.Errorf("check white list failed, request ip is bad")
		return
	}

	requestIp := net.ParseIP(requestIp0)

	if requestIp == nil {
		err = fmt.Errorf("check white list failed, request ip is bad")
		return
	}

	if w.Ips == nil || len(w.Ips) == 0 {
		err = fmt.Errorf("no ips in white list")
		return
	}

	for _, ip := range w.Ips {
		if ip.Contains(requestIp) {
			succeed = true
			break
		}
	}

	return
}

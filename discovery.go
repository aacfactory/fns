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
	"github.com/valyala/fasthttp"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type DiscoveryConfig struct {
	Enable bool            `json:"enable,omitempty"`
	Kind   string          `json:"kind,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

var discoveryRetrieverMap map[string]DiscoveryRetriever = nil

type DiscoveryRetriever func(options DiscoveryOption) (d Discovery, err error)

//RegisterDiscoveryRetriever 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterDiscoveryRetriever(kind string, retriever DiscoveryRetriever) {
	discoveryRetrieverMap[kind] = retriever
}

// +-------------------------------------------------------------------------------------------------------------------+

// Registration
// 注册结果
type Registration struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	ClientTLS ClientTLS `json:"clientTls"`
}

type DiscoveryOption struct {
	Address   string
	ClientTLS ClientTLS
	Config    []byte
}

// Discovery
// Fn的注册与发现
type Discovery interface {
	// Publish
	// 注册Fn
	Publish(name string) (registrationId string, err error)
	// UnPublish
	// 注销Fn
	UnPublish(registrationId string) (err error)
	// Get
	// 发现Fn注册信息
	Get(name string) (registrations []Registration, err error)
	// SyncRegistrations
	// 同步Registrations，类似Watching
	SyncRegistrations() (ch <-chan map[string][]Registration, err error)
	// Close
	// 关闭
	Close()
}

func discoveryRegistrationMapToHttpClient(registrations []Registration) (client FnHttpClient, err error) {
	lb := &fasthttp.LBClient{
		Clients: make([]fasthttp.BalancingClient, 0, len(registrations)),
		Timeout: 30 * time.Second,
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

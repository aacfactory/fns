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
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
)

const (
	proxyHandlerName = "proxy"
)

func newProxyHandler(registrations *Registrations, deployedCh <-chan map[string]*endpoint, dialer HttpClientDialer, openApiVersion string, devMode bool) (handler *proxyHandler) {
	ph := &proxyHandler{
		appId:         "",
		appName:       "",
		appVersion:    versions.Version{},
		log:           nil,
		devMode:       devMode,
		endpoints:     nil,
		registrations: registrations,
		dialer:        nil,
		discovery:     nil,
	}
	go func(handler *proxyHandler, deployedCh <-chan map[string]*endpoint, openApiVersion string, dialer HttpClientDialer) {
		eps, ok := <-deployedCh
		if !ok {
			return
		}
		if eps == nil || len(eps) == 0 {
			return
		}
		names := make([]string, 0, 1)
		namesWithInternal := make([]string, 0, 1)
		documents := make(map[string]Document)
		for name, ep := range eps {
			namesWithInternal = append(namesWithInternal, name)
			if !ep.Internal() {
				names = append(names, name)
				document := ep.Document()
				if document != nil {
					documents[name] = document
				}
			}
		}
		namesBytes, namesErr := json.Marshal(names)
		if namesErr == nil {
			handler.names = namesBytes
		}
		namesWithInternalBytes, namesWithInternalErr := json.Marshal(namesWithInternal)
		if namesWithInternalErr == nil {
			handler.namesWithInternal = namesWithInternalBytes
		}
		document, documentErr := encodeDocuments(handler.appId, handler.appName, handler.appVersion, eps)
		if documentErr == nil {
			handler.documents = document
		}
		openapi, openapiErr := encodeOpenapi(openApiVersion, handler.appId, handler.appName, handler.appVersion, eps)
		if openapiErr == nil {
			handler.openapi = openapi
		}
	}(ph, deployedCh, openApiVersion, dialer)
	handler = ph
	return
}

type proxyHandler struct {
	appId         string
	appName       string
	appVersion    versions.Version
	log           logs.Logger
	devMode       bool
	names         []string
	endpoints     map[string]*endpoint
	registrations *Registrations
	dialer        HttpClientDialer
	discovery     EndpointDiscovery
}

func (proxy *proxyHandler) Name() (name string) {
	name = proxyHandlerName
	return
}

func (proxy *proxyHandler) Build(options *HttpHandlerOptions) (err error) {
	proxy.log = options.Log
	proxy.appId = options.AppId
	proxy.appName = options.AppName
	proxy.appVersion = options.AppVersion
	proxy.discovery = options.Discovery
	return
}

func (proxy *proxyHandler) Accept(request *http.Request) (ok bool) {
	// todo:
	// handle dispatch (when devMode enabled, support internal request, else not support)
	// handle /services/names (only devMode enabled, and get nodeId from httpProxyTargetNodeId)
	// handle /services/documents
	// handle /services/openapi
	// handle /cluster/nodes (only devMode enabled, return nodes, so cluster devMode should be {ClusterProxy}, and use it to get nodes)
	// -- or Cluster is ClusterProxy, no DevMode, use DevMode to create ClusterProxy, use ClusterProxy to get nodes
	return
}

func (proxy *proxyHandler) Close() {

	return
}

func (proxy *proxyHandler) ServeHTTP(writer http.ResponseWriter, r *http.Request) {

	return
}

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

import "net/http"

const (
	proxyHandlerName = "proxy"
)

func newProxyHandler(devMode bool) (handler *proxyHandler) {

	return
}

type proxyHandler struct {
}

func (proxy *proxyHandler) Name() (name string) {
	name = proxyHandlerName
	return
}

func (proxy *proxyHandler) Build(options *HttpHandlerOptions) (err error) {

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

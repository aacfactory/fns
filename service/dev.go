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
	devProxyHandlerName = "dev"
)

// TODO: 负责集群环境下的开发方式，通过代理转发的方式实现，
// * internal request
// * cluster （join，leave，nodes，shared）
// * /services/names
type devProxyHandler struct {
	cluster       Cluster
	registrations *Registrations
}

func (handler *devProxyHandler) Name() (name string) {
	name = devProxyHandlerName
	return
}

func (handler *devProxyHandler) Build(options TransportHandlerOptions) (err error) {

	return
}

func (handler *devProxyHandler) Accept(r *http.Request) (ok bool) {

	return
}

func (handler *devProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	return
}

func (handler *devProxyHandler) Close() {
	return
}

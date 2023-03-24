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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"strings"
	"time"
)

const (
	proxyHandlerName = "proxy"
)

func newProxyHandler(registrations *Registrations, deployedCh <-chan map[string]*endpoint, dialer HttpClientDialer, openApiVersion string, devMode bool, secretKey []byte) (handler *proxyHandler) {
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
		signer:        secret.NewSigner(secretKey),
	}
	go func(handler *proxyHandler, deployedCh <-chan map[string]*endpoint, openApiVersion string, dialer HttpClientDialer) {
		eps, ok := <-deployedCh
		if !ok {
			return
		}
		if eps == nil || len(eps) == 0 {
			return
		}
		handler.endpoints = eps
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
	endpoints     map[string]*endpoint
	registrations *Registrations
	dialer        HttpClientDialer
	discovery     EndpointDiscovery
	signer        *secret.Signer
}

func (handler *proxyHandler) Name() (name string) {
	name = proxyHandlerName
	return
}

func (handler *proxyHandler) Build(options *HttpHandlerOptions) (err error) {
	handler.log = options.Log
	handler.appId = options.AppId
	handler.appName = options.AppName
	handler.appVersion = options.AppVersion
	handler.discovery = options.Discovery
	return
}

func (handler *proxyHandler) Accept(r *http.Request) (ok bool) {
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/documents" && r.URL.Query().Get("native") == ""
	if ok {
		return
	}
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/openapi" && r.URL.Query().Get("native") == ""
	if ok {
		return
	}
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/names" && r.URL.Query().Get("native") == ""
	if ok {
		return
	}
	pathItems := strings.Split(r.URL.Path, "/")
	ok = r.Method == http.MethodPost && r.Header.Get(httpContentType) == httpContentTypeJson && len(pathItems) == 3
	if ok {
		// local service request
		serviceName := pathItems[0]
		if _, has := handler.endpoints[serviceName]; has {
			ok = false
			return
		}
	}
	return
}

func (handler *proxyHandler) Close() {

	return
}

func (handler *proxyHandler) ServeHTTP(writer http.ResponseWriter, r *http.Request) {

	// handle /services/names (when devMode enabled, then try to get nodeId from httpProxyTargetNodeId)
	return
}

func (handler *proxyHandler) handleNames(writer http.ResponseWriter, r *http.Request) {

	return
}

func (handler *proxyHandler) handleOpenapi(writer http.ResponseWriter, r *http.Request) {
	// use ?native=true to fetch
	return
}

func (handler *proxyHandler) handleDocuments(writer http.ResponseWriter, r *http.Request) {
	// use ?native=true to fetch
	return
}

func (handler *proxyHandler) handleProxy(writer http.ResponseWriter, r *http.Request) {
	pathItems := strings.Split(r.URL.Path, "/")
	if len(pathItems) != 3 {
		handler.failed(writer, "", 0, http.StatusBadRequest, errors.Warning("fns: invalid request url path"))
		return
	}
	if r.Header.Get(httpDeviceIdHeader) == "" {
		handler.failed(writer, "", 0, http.StatusBadRequest, errors.Warning("fns: X-Fns-Device-Id is required"))
		return
	}
	if r.Header.Get(httpProxyTargetNodeId) != "" {
		handler.handleDevProxy(writer, r)
		return
	}
	// todo
	return
}

func (handler *proxyHandler) handleDevProxy(writer http.ResponseWriter, r *http.Request) {
	// todo verify signature
	return
}

func (handler *proxyHandler) succeed(writer http.ResponseWriter, id string, latency time.Duration, result interface{}) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	if id != "" {
		writer.Header().Set(httpRequestIdHeader, id)
	}
	if handler.log.DebugEnabled() {
		writer.Header().Set(httpHandleLatencyHeader, latency.String())
	}
	writer.WriteHeader(http.StatusOK)
	body, encodeErr := json.Marshal(result)
	if encodeErr != nil {
		cause := errors.Warning("encode result failed").WithCause(encodeErr)
		handler.failed(writer, id, latency, cause.Code(), cause)
		return
	}
	n := 0
	bodyLen := len(body)
	for n < bodyLen {
		nn, writeErr := writer.Write(body[n:])
		if writeErr != nil {
			return
		}
		n += nn
	}
	return
}

func (handler *proxyHandler) failed(writer http.ResponseWriter, id string, latency time.Duration, status int, cause interface{}) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	if id != "" {
		writer.Header().Set(httpRequestIdHeader, id)
	}
	if handler.log.DebugEnabled() {
		writer.Header().Set(httpHandleLatencyHeader, latency.String())
	}
	writer.WriteHeader(status)
	body, _ := json.Marshal(cause)
	n := 0
	bodyLen := len(body)
	for n < bodyLen {
		nn, writeErr := writer.Write(body[n:])
		if writeErr != nil {
			return
		}
		n += nn
	}
	return
}

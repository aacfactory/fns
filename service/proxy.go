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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/lru"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"golang.org/x/sync/singleflight"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	proxyHandlerName = "proxy"
)

func newProxyHandler(cluster Cluster, registrations *Registrations, deployedCh <-chan map[string]*endpoint, dialer HttpClientDialer, openApiVersion string, devMode bool, secretKey []byte) (handler *proxyHandler) {
	handler = &proxyHandler{
		appId:          "",
		appName:        "",
		appVersion:     versions.Version{},
		openApiVersion: openApiVersion,
		log:            nil,
		ready:          false,
		devMode:        devMode,
		cluster:        cluster,
		endpoints:      make(map[string]*endpoint),
		registrations:  registrations,
		dialer:         dialer,
		discovery:      nil,
		signer:         secret.NewSigner(secretKey),
		cache:          lru.New[string, json.RawMessage](8),
		group:          &singleflight.Group{},
	}
	go func(handler *proxyHandler, deployedCh <-chan map[string]*endpoint, dialer HttpClientDialer) {
		eps, ok := <-deployedCh
		handler.ready = true
		if !ok {
			return
		}
		if eps == nil || len(eps) == 0 {
			return
		}
		handler.endpoints = eps
	}(handler, deployedCh, dialer)
	return
}

type proxyHandler struct {
	appId          string
	appName        string
	appVersion     versions.Version
	openApiVersion string
	log            logs.Logger
	ready          bool
	devMode        bool
	cluster        Cluster
	endpoints      map[string]*endpoint
	registrations  *Registrations
	dialer         HttpClientDialer
	discovery      EndpointDiscovery
	signer         *secret.Signer
	cache          *lru.LRU[string, json.RawMessage]
	group          *singleflight.Group
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
	if !handler.ready {
		handler.failed(writer, "", 0, http.StatusTooEarly, errors.Warning("proxy: handler is not ready, try later again").WithMeta("handler", handler.Name()))
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/names" {
		handler.handleNames(writer, r)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/openapi" {
		handler.handleOpenapi(writer, r)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/documents" {
		handler.handleDocuments(writer, r)
		return
	}
	handler.handleProxy(writer, r)
	return
}

func (handler *proxyHandler) handleNames(writer http.ResponseWriter, r *http.Request) {
	const (
		namesKey        = "names"
		refreshGroupKey = "names_refresh"
	)
	handleBegAT := time.Time{}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	groupKey := namesKey
	if r.URL.Query().Get("refresh") == "true" {
		groupKey = refreshGroupKey
	}
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		if r.URL.Query().Get("refresh") != "true" {
			cached, has := handler.cache.Get(namesKey)
			if has {
				v = cached
				return
			}
		}
		nodes, getNodesErr := listMembers(r.Context(), handler.cluster, handler.appId, handler.appName, handler.appVersion)
		if getNodesErr != nil {
			err = errors.Warning("proxy: handle names request failed").WithCause(getNodesErr)
			return
		}
		names := make([]string, 0, 1)
		for name := range handler.endpoints {
			names = append(names, name)
		}
		memberNodeNames := make(map[string]int)
		for _, node := range nodes {
			nodeNames, nodeNamesErr := listMemberServiceNames(r.Context(), node, handler.dialer, handler.appId, handler.signer)
			if nodeNamesErr != nil {
				err = errors.Warning("proxy: handle names request failed").WithCause(nodeNamesErr)
				return
			}
			if nodeNames != nil && len(nodeNames) > 0 {
				for _, name := range nodeNames {
					memberNodeNames[name] = 1
				}
			}
		}
		for name := range memberNodeNames {
			names = append(names, name)
		}
		p, encodeErr := json.Marshal(names)
		if encodeErr != nil {
			err = errors.Warning("proxy: handle names request failed").WithCause(encodeErr)
			return
		}
		handler.cache.Add(namesKey, p, 60*time.Second)
		v = json.RawMessage(p)
		return
	})
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	if err != nil {
		handler.failed(writer, "", latency, 555, errors.Map(err))
		return
	}
	handler.succeed(writer, "", latency, v)
	return
}

func (handler *proxyHandler) handleOpenapi(writer http.ResponseWriter, r *http.Request) {
	const (
		namesKey        = "openapi"
		refreshGroupKey = "openapi_refresh"
	)
	handleBegAT := time.Time{}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	groupKey := namesKey
	if r.URL.Query().Get("refresh") == "true" {
		groupKey = refreshGroupKey
	}
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		if r.URL.Query().Get("refresh") != "true" {
			cached, has := handler.cache.Get(namesKey)
			if has {
				v = cached
				return
			}
		}
		openapi := newOpenapi(handler.openApiVersion, handler.appId, handler.appName, handler.appVersion, handler.endpoints)
		nodes, getNodesErr := listMembers(r.Context(), handler.cluster, handler.appId, handler.appName, handler.appVersion)
		if getNodesErr != nil {
			err = errors.Warning("proxy: handle openapi request failed").WithCause(getNodesErr)
			return
		}
		for _, node := range nodes {
			memberOpenapi, memberOpenapiErr := getMemberOpenapi(r.Context(), node, handler.dialer)
			if memberOpenapiErr != nil {
				err = errors.Warning("proxy: handle openapi request failed").WithCause(memberOpenapiErr)
				return
			}
			openapi.Merge(memberOpenapi)
		}
		p, encodeErr := json.Marshal(openapi)
		if encodeErr != nil {
			err = errors.Warning("proxy: handle openapi request failed").WithCause(encodeErr)
			return
		}
		handler.cache.Add(namesKey, p, 60*time.Second)
		v = json.RawMessage(p)
		return
	})
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	if err != nil {
		handler.failed(writer, "", latency, 555, errors.Map(err))
		return
	}
	handler.succeed(writer, "", latency, v)
	return
}

func (handler *proxyHandler) handleDocuments(writer http.ResponseWriter, r *http.Request) {
	const (
		namesKey        = "documents"
		refreshGroupKey = "documents_refresh"
	)
	handleBegAT := time.Time{}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	groupKey := namesKey
	if r.URL.Query().Get("refresh") == "true" {
		groupKey = refreshGroupKey
	}
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		if r.URL.Query().Get("refresh") != "true" {
			cached, has := handler.cache.Get(namesKey)
			if has {
				v = cached
				return
			}
		}
		documents, documentsErr := newDocuments(handler.endpoints, handler.appVersion)
		if documentsErr == nil {
			err = errors.Warning("proxy: handle documents request failed").WithCause(documentsErr)
			return
		}
		nodes, getNodesErr := listMembers(r.Context(), handler.cluster, handler.appId, handler.appName, handler.appVersion)
		if getNodesErr != nil {
			err = errors.Warning("proxy: handle documents request failed").WithCause(getNodesErr)
			return
		}
		for _, node := range nodes {
			memberDocuments, memberDocumentsErr := getMemberDocument(r.Context(), node, handler.dialer)
			if memberDocumentsErr != nil {
				err = errors.Warning("proxy: handle documents request failed").WithCause(memberDocumentsErr)
				return
			}
			documents = documents.Merge(memberDocuments)
		}
		p, encodeErr := json.Marshal(documents)
		if encodeErr != nil {
			err = errors.Warning("proxy: handle documents request failed").WithCause(encodeErr)
			return
		}
		handler.cache.Add(namesKey, p, 60*time.Second)
		v = json.RawMessage(p)
		return
	})
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	if err != nil {
		handler.failed(writer, "", latency, 555, errors.Map(err))
		return
	}
	handler.succeed(writer, "", latency, v)
	return
}

func (handler *proxyHandler) handleProxy(writer http.ResponseWriter, r *http.Request) {
	pathItems := strings.Split(r.URL.Path, "/")
	if len(pathItems) != 3 {
		handler.failed(writer, "", 0, 555, errors.Warning("proxy: invalid request url path"))
		return
	}
	if r.Header.Get(httpDeviceIdHeader) == "" {
		handler.failed(writer, "", 0, 555, errors.Warning("proxy: X-Fns-Device-Id is required"))
		return
	}
	if r.Header.Get(httpProxyTargetNodeId) != "" {
		handler.handleDevProxy(writer, r)
		return
	}
	leftVersion, rightVersion, parseErr := versions.ParseRange(r.Header.Get(httpRequestVersionsHeader))
	if parseErr != nil {
		handler.failed(writer, "", 0, 555, errors.Warning("proxy: X-Fns-Request-Version is invalid").WithCause(parseErr))
		return
	}
	serviceName := pathItems[1]
	registration, has := handler.registrations.Get(serviceName, leftVersion, rightVersion)
	if !has {
		handler.failed(writer, "", 0, 404, errors.Warning("proxy: service was not found").WithMeta("service", serviceName))
		return
	}
	client, clientErr := handler.dialer.Dial(registration.address)
	if clientErr != nil {
		handler.failed(writer, "", 0, 555, errors.Warning("proxy: get host dialer failed").WithMeta("service", serviceName).WithCause(clientErr))
		return
	}
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		if readBodyErr != io.EOF {
			handler.failed(writer, "", 0, 555, errors.Warning("proxy: read body failed").WithCause(readBodyErr))
			return
		}
	}
	status, header, respBody, postErr := client.Post(r.Context(), r.URL.Path, r.Header.Clone(), body)
	if postErr != nil {
		handler.failed(writer, "", 0, 555, errors.Warning("proxy: proxy failed").WithMeta("service", serviceName).WithCause(postErr))
		return
	}
	writer.WriteHeader(status)
	if header != nil && len(header) > 0 {
		for k, vv := range header {
			for _, v := range vv {
				writer.Header().Add(k, v)
			}
		}
	}
	n := 0
	bodyLen := len(respBody)
	for n < bodyLen {
		nn, writeErr := writer.Write(respBody[n:])
		if writeErr != nil {
			return
		}
		n += nn
	}
	return
}

func (handler *proxyHandler) handleDevProxy(writer http.ResponseWriter, r *http.Request) {
	if !handler.devMode {
		handler.failed(writer, "", 0, 555, errors.Warning("proxy: dev mode is not enabled"))
		return
	}
	nodeId := r.Header.Get(httpProxyTargetNodeId)
	pathItems := strings.Split(r.URL.Path, "/")
	serviceName := pathItems[1]
	registration, has := handler.registrations.GetExact(serviceName, nodeId)
	if !has {
		handler.failed(writer, "", 0, 555, errors.Warning("proxy: host was not found").WithMeta("service", serviceName).WithMeta("id", nodeId))
		return
	}
	client, clientErr := handler.dialer.Dial(registration.address)
	if clientErr != nil {
		handler.failed(writer, "", 0, 555, errors.Warning("proxy: get host dialer failed").WithMeta("service", serviceName).WithMeta("id", nodeId).WithCause(clientErr))
		return
	}

	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		if readBodyErr != io.EOF {
			handler.failed(writer, "", 0, 555, errors.Warning("proxy: read body failed").WithCause(readBodyErr))
			return
		}
	}
	// verify signature
	if !handler.signer.Verify(body, bytex.FromString(r.Header.Get(httpRequestSignatureHeader))) {
		handler.failed(writer, "", 0, http.StatusNotAcceptable, errors.Warning("proxy: X-Fns-Request-Signature is invalid"))
		return
	}
	status, header, respBody, postErr := client.Post(r.Context(), r.URL.Path, r.Header.Clone(), body)
	if postErr != nil {
		handler.failed(writer, "", 0, 555, errors.Warning("proxy: proxy failed").WithMeta("service", serviceName).WithMeta("id", nodeId).WithCause(postErr))
		return
	}
	writer.WriteHeader(status)
	if header != nil && len(header) > 0 {
		for k, vv := range header {
			for _, v := range vv {
				writer.Header().Add(k, v)
			}
		}
	}
	n := 0
	bodyLen := len(respBody)
	for n < bodyLen {
		nn, writeErr := writer.Write(respBody[n:])
		if writeErr != nil {
			return
		}
		n += nn
	}
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

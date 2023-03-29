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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/documents"
	"github.com/aacfactory/fns/service/internal/lru"
	"github.com/aacfactory/fns/service/internal/ratelimit"
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

type ProxyHandlerOptions struct {
	Signer        *secret.Signer
	Registrations *Registrations
	DeployedCh    <-chan map[string]*endpoint
}

// todo proxy 单独做，单独拥有一个http server，与service的独立，然后发布出去的只有service的，cluster里取所有的port，然后由registation般配port是否为service。
// todo 那websocket怎么办。。。
// todo 主要是ssl的问题，
// caseA: 两个server，只有一个service handler，proxy handler不要了，在service handler里增加cluster特性，两个server共享一个handler？？？proxy的handler是全的，http里的是只有services。
// todo Registrations 不管 dev，由Registrations的dialer处理dev，所以dialer需要代理
func newProxyHandler(options ProxyHandlerOptions) (handler *proxyHandler) {
	handler = &proxyHandler{
		log:                    nil,
		ready:                  false,
		documents:              documents.NewDocuments(),
		disableHandleDocuments: false,
		disableHandleOpenapi:   false,
		openapiVersion:         "",
		appId:                  "",
		appName:                "",
		appVersion:             versions.Version{},
		signer:                 options.Signer,
		devMode:                false,
		registrations:          options.Registrations,
		attachments:            nil,
		group:                  &singleflight.Group{},
		limiter:                nil,
		retryAfter:             "",
	}
	go func(handler *proxyHandler, deployedCh <-chan map[string]*endpoint) {
		_, ok := <-deployedCh
		if !ok {
			return
		}
		handler.ready = true
	}(handler, options.DeployedCh)
	return
}

type proxyHandlerConfig struct {
	DisableHandleDocuments bool                 `json:"disableHandleDocuments"`
	DisableHandleOpenapi   bool                 `json:"disableHandleOpenapi"`
	OpenapiVersion         string               `json:"openapiVersion"`
	Limiter                RequestLimiterConfig `json:"limiter"`
}

type proxyHandler struct {
	log                    logs.Logger
	ready                  bool
	documents              documents.Documents
	disableHandleDocuments bool
	disableHandleOpenapi   bool
	openapiVersion         string
	appId                  string
	appName                string
	appVersion             versions.Version
	signer                 *secret.Signer
	devMode                bool
	registrations          *Registrations
	attachments            *lru.LRU[string, []byte]
	group                  *singleflight.Group
	limiter                *ratelimit.Limiter
	retryAfter             string
}

func (handler *proxyHandler) Name() (name string) {
	name = proxyHandlerName
	return
}

func (handler *proxyHandler) Build(options *HttpHandlerOptions) (err error) {
	config := proxyHandlerConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: build proxy handler failed").WithCause(configErr)
		return
	}
	handler.log = options.Log
	handler.appId = options.AppId
	handler.appName = options.AppName
	handler.appVersion = options.AppVersion
	handler.disableHandleDocuments = config.DisableHandleDocuments
	handler.disableHandleOpenapi = config.DisableHandleOpenapi
	if !handler.disableHandleOpenapi {
		handler.openapiVersion = strings.TrimSpace(config.OpenapiVersion)
	}
	maxPerDeviceRequest := config.Limiter.MaxPerDeviceRequest
	if maxPerDeviceRequest < 1 {
		maxPerDeviceRequest = 8
	}
	retryAfter := 10 * time.Second
	if config.Limiter.RetryAfter != "" {
		retryAfter, err = time.ParseDuration(strings.TrimSpace(config.Limiter.RetryAfter))
		if err != nil {
			err = errors.Warning("fns: build services handler failed").WithCause(errors.Warning("retryAfter must be time.Duration format").WithCause(err))
			return
		}
	}
	handler.limiter = ratelimit.New(maxPerDeviceRequest)
	handler.retryAfter = fmt.Sprintf("%d", int(retryAfter/time.Second))
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
	if handler.cache != nil {
		handler.cache.Close()
	}
	return
}

func (handler *proxyHandler) ServeHTTP(writer http.ResponseWriter, r *http.Request) {
	if !handler.ready {
		handler.failed(writer, errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: handler is not ready, try later again").WithMeta("handler", handler.Name()))
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
	groupKey := namesKey
	if r.URL.Query().Get("refresh") == "true" {
		groupKey = refreshGroupKey
	}
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		if r.URL.Query().Get("refresh") != "true" {
			cached, has := handler.attachment.Get(namesKey)
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
		handler.attachment.Add(namesKey, p, 60*time.Second)
		v = p
		return
	})
	if err != nil {
		handler.failed(writer, errors.Map(err))
		return
	}
	handler.succeed(writer, v.([]byte))
	return
}

func (handler *proxyHandler) handleOpenapi(writer http.ResponseWriter, r *http.Request) {
	const (
		namesKey        = "openapi"
		refreshGroupKey = "openapi_refresh"
	)
	groupKey := namesKey
	if r.URL.Query().Get("refresh") == "true" {
		groupKey = refreshGroupKey
	}
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		if r.URL.Query().Get("refresh") != "true" {
			cached, has := handler.attachment.Get(namesKey)
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
		handler.attachment.Add(namesKey, p, 60*time.Second)
		v = p
		return
	})
	if err != nil {
		handler.failed(writer, errors.Map(err))
		return
	}
	handler.succeed(writer, v.([]byte))
	return
}

func (handler *proxyHandler) handleDocuments(writer http.ResponseWriter, r *http.Request) {
	const (
		namesKey        = "documents"
		refreshGroupKey = "documents_refresh"
	)
	groupKey := namesKey
	if r.URL.Query().Get("refresh") == "true" {
		groupKey = refreshGroupKey
	}
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		if r.URL.Query().Get("refresh") != "true" {
			cached, has := handler.attachment.Get(namesKey)
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
		handler.attachment.Add(namesKey, p, 60*time.Second)
		v = p
		return
	})
	if err != nil {
		handler.failed(writer, errors.Map(err))
		return
	}
	handler.succeed(writer, v.([]byte))
	return
}

func (handler *proxyHandler) handleProxy(writer http.ResponseWriter, r *http.Request) {
	pathItems := strings.Split(r.URL.Path, "/")
	if len(pathItems) != 3 {
		handler.failed(writer, errors.Warning("proxy: invalid request url path"))
		return
	}
	if r.Header.Get(httpDeviceIdHeader) == "" {
		handler.failed(writer, errors.Warning("proxy: X-Fns-Device-Id is required"))
		return
	}
	if r.Header.Get(httpDevModeHeader) != "" {
		handler.handleDevProxy(writer, r)
		return
	}
	rvs, hasVersion, parseVersionErr := ParseRequestVersionFromHeader(r.Header)
	if parseVersionErr != nil {
		handler.failed(writer, errors.Warning("proxy: X-Fns-Request-Version is invalid").WithCause(parseVersionErr))
		return
	}
	if !hasVersion {
		rvs = AllowAllRequestVersions()
	}
	serviceName := pathItems[1]

	registration, has := handler.registrations.Get(serviceName, rvs)
	if !has {
		handler.failed(writer, errors.Warning("proxy: service was not found").WithMeta("service", serviceName))
		return
	}

	client, clientErr := handler.dialer.Dial(registration.address)
	if clientErr != nil {
		handler.failed(writer, errors.Warning("proxy: get host dialer failed").WithMeta("service", serviceName).WithCause(clientErr))
		return
	}
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		handler.failed(writer, errors.Warning("proxy: read body failed").WithCause(readBodyErr))
		return
	}
	_ = r.Body.Close()

	if handler.cache != nil {
		if handler.cache.Cached(writer, r.Header, r.URL, rvs, body) {
			return
		}
	}
	status, header, respBody, postErr := client.Post(r.Context(), r.URL.Path, r.Header.Clone(), body)
	if postErr != nil {
		handler.failed(writer, errors.Warning("proxy: proxy failed").WithMeta("service", serviceName).WithCause(postErr))
		return
	}
	if status == http.StatusNotModified {

	}
	// todo cache
	writer.WriteHeader(status)
	if header != nil && len(header) > 0 {
		for k, vv := range header {
			for _, v := range vv {
				writer.Header().Add(k, v)
			}
		}
	}
	if status == http.StatusOK && handler.cache != nil {
		handler.cache.TrySet(writer, header, r.URL, rvs, respBody)
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
		handler.failed(writer, errors.Warning("proxy: dev mode is not enabled"))
		return
	}
	nodeId := r.Header.Get(httpDevModeHeader)
	pathItems := strings.Split(r.URL.Path, "/")
	serviceName := pathItems[1]
	registration, has := handler.registrations.GetExact(serviceName, nodeId)
	if !has {
		handler.failed(writer, errors.NotFound("proxy: host was not found").WithMeta("service", serviceName).WithMeta("id", nodeId))
		return
	}
	client, clientErr := handler.dialer.Dial(registration.address)
	if clientErr != nil {
		handler.failed(writer, errors.Warning("proxy: get host dialer failed").WithMeta("service", serviceName).WithMeta("id", nodeId).WithCause(clientErr))
		return
	}

	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		handler.failed(writer, errors.Warning("proxy: read body failed").WithCause(readBodyErr))
		return
	}
	_ = r.Body.Close()

	// verify signature
	if !handler.signer.Verify(body, bytex.FromString(r.Header.Get(httpRequestSignatureHeader))) {
		handler.failed(writer, errors.Warning("proxy: X-Fns-Request-Signature is invalid"))
		return
	}
	status, header, respBody, postErr := client.Post(r.Context(), r.URL.Path, r.Header.Clone(), body)
	if postErr != nil {
		handler.failed(writer, errors.Warning("proxy: proxy failed").WithMeta("service", serviceName).WithMeta("id", nodeId).WithCause(postErr))
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

func (handler *proxyHandler) succeed(writer http.ResponseWriter, body []byte) {
	writer.WriteHeader(http.StatusOK)
	writer.Header().Set(httpContentType, httpContentTypeJson)
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

func (handler *proxyHandler) failed(writer http.ResponseWriter, cause errors.CodeError) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	status := cause.Code()
	if status == 0 {
		status = 555
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

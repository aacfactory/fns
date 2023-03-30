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
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/documents"
	"github.com/aacfactory/fns/service/internal/lru"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"golang.org/x/sync/singleflight"
	"net/http"
	"strings"
	"time"
)

const (
	proxyHandlerName = "proxy"
	proxyContextKey  = "@fns_proxy"
)

type ProxyHandlerOptions struct {
	DevMode       bool
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
		disableHandleDocuments: false,
		disableHandleOpenapi:   false,
		openapiVersion:         "",
		appId:                  "",
		appName:                "",
		appVersion:             versions.Version{},
		signer:                 options.Signer,
		devMode:                options.DevMode,
		registrations:          options.Registrations,
		attachments:            lru.New[string, json.RawMessage](8),
		group:                  &singleflight.Group{},
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
	DocumentsTTL           string               `json:"documentsTTL"`
	Limiter                RequestLimiterConfig `json:"limiter"`
}

type proxyHandler struct {
	log                    logs.Logger
	ready                  bool
	disableHandleDocuments bool
	disableHandleOpenapi   bool
	openapiVersion         string
	documentsTTL           time.Duration
	appId                  string
	appName                string
	appVersion             versions.Version
	signer                 *secret.Signer
	devMode                bool
	registrations          *Registrations
	attachments            *lru.LRU[string, json.RawMessage]
	group                  *singleflight.Group
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
	documentsTTL := 60 * time.Second
	if config.DocumentsTTL != "" {
		documentsTTL, err = time.ParseDuration(strings.TrimSpace(config.DocumentsTTL))
		if err != nil {
			err = errors.Warning("fns: build services handler failed").WithCause(errors.Warning("documentsTTL must be time.Duration format").WithCause(err))
			return
		}
	}
	if documentsTTL < 1 {
		documentsTTL = 60 * time.Second
	}
	handler.documentsTTL = documentsTTL
	maxPerDeviceRequest := config.Limiter.MaxPerDeviceRequest
	if maxPerDeviceRequest < 1 {
		maxPerDeviceRequest = 8
	}
	return
}

func (handler *proxyHandler) Accept(r *http.Request) (ok bool) {
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/documents"
	if ok {
		return
	}
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/openapi"
	if ok {
		return
	}
	pathItems := strings.Split(r.URL.Path, "/")
	ok = r.Method == http.MethodPost && r.Header.Get(httpContentType) == httpContentTypeJson && len(pathItems) == 3
	return
}

func (handler *proxyHandler) Close() {
	return
}

func (handler *proxyHandler) ServeHTTP(writer http.ResponseWriter, r *http.Request) {
	if !handler.ready {
		handler.failed(writer, errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: handler is not ready, try later again").WithMeta("handler", handler.Name()))
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
	if r.Header.Get(httpDevModeHeader) == "" {
		handler.handleProxy(writer, r)
	} else {
		handler.handleDevProxy(writer, r)
	}

	return
}

func (handler *proxyHandler) fetchDocuments() (v documents.Documents, err error) {
	value, fetchErr, _ := handler.group.Do("documents:fetch", func() (v interface{}, err error) {
		doc, fetchErr := handler.registrations.FetchDocuments()
		if fetchErr != nil {
			err = fetchErr
			return
		}
		v = doc
		return
	})
	if fetchErr != nil {
		err = fetchErr
		return
	}
	v = (value).(documents.Documents)
	return
}

func (handler *proxyHandler) handleOpenapi(writer http.ResponseWriter, r *http.Request) {
	version := versions.Latest()
	if targetVersion := r.URL.Query().Get("version"); targetVersion != "" {
		var err error
		version, err = versions.Parse(targetVersion)
		if err != nil {
			handler.failed(writer, errors.Warning("proxy: parse version failed").WithCause(err))
			return
		}
	}
	handleBegAT := time.Time{}
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	key := fmt.Sprintf("openapi:%s", version.String())
	refresh := r.URL.Query().Get("refresh") == "true"
	v, err, _ := handler.group.Do(fmt.Sprintf("%s:%v", key, refresh), func() (v interface{}, err error) {
		if !refresh {
			cached, has := handler.attachments.Get(key)
			if has {
				v = cached
				return
			}
		}
		doc, fetchDocumentsErr := handler.fetchDocuments()
		if fetchDocumentsErr != nil {
			err = errors.Warning("proxy: fetch backend documents failed").WithCause(fetchDocumentsErr)
			return
		}
		api := doc.Openapi(handler.openapiVersion, handler.appId, handler.appName, version)
		p, encodeErr := json.Marshal(api)
		if encodeErr != nil {
			err = errors.Warning("proxy: encode openapi failed").WithCause(encodeErr)
			return
		}
		handler.attachments.Add(key, p, handler.documentsTTL)
		v = json.RawMessage(p)
		return
	})
	if err != nil {
		handler.failed(writer, errors.Map(err))
		return
	}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	handler.succeed(writer, "", latency, v.(json.RawMessage))
	return
}

func (handler *proxyHandler) handleDocuments(writer http.ResponseWriter, r *http.Request) {
	handleBegAT := time.Time{}
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	key := "documents:write"
	refresh := r.URL.Query().Get("refresh") == "true"
	v, err, _ := handler.group.Do(fmt.Sprintf("%s:%v", key, refresh), func() (v interface{}, err error) {
		if !refresh {
			cached, has := handler.attachments.Get(key)
			if has {
				v = cached
				return
			}
		}
		doc, fetchDocumentsErr := handler.fetchDocuments()
		if fetchDocumentsErr != nil {
			err = errors.Warning("proxy: fetch backend documents failed").WithCause(fetchDocumentsErr)
			return
		}
		p, encodeErr := json.Marshal(doc)
		if encodeErr != nil {
			err = errors.Warning("proxy: encode documents failed").WithCause(encodeErr)
			return
		}
		handler.attachments.Add(key, p, handler.documentsTTL)
		v = json.RawMessage(p)
		return
	})
	if err != nil {
		handler.failed(writer, errors.Map(err))
		return
	}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	handler.succeed(writer, "", latency, v.(json.RawMessage))
	return
}

func (handler *proxyHandler) handleProxy(writer http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	ctx = context.WithValue(ctx, proxyContextKey, 1)

	return
}

func (handler *proxyHandler) handleDevProxy(writer http.ResponseWriter, r *http.Request) {

	return
}

func (handler *proxyHandler) succeed(writer http.ResponseWriter, id string, latency time.Duration, result interface{}) {
	body, encodeErr := json.Marshal(result)
	if encodeErr != nil {
		cause := errors.Warning("encode result failed").WithCause(encodeErr)
		handler.failed(writer, cause)
		return
	}
	if id != "" {
		writer.Header().Set(httpRequestIdHeader, id)
	}
	if handler.log.DebugEnabled() {
		writer.Header().Set(httpHandleLatencyHeader, latency.String())
	}
	handler.write(writer, http.StatusOK, body)
	return
}

func (handler *proxyHandler) failed(writer http.ResponseWriter, cause errors.CodeError) {
	if cause == nil {
		handler.write(writer, 555, bytex.FromString(emptyJson))
		return
	}
	status := cause.Code()
	if status == 0 {
		status = 555
	}
	body, _ := json.Marshal(cause)
	handler.write(writer, status, body)
	return
}

func (handler *proxyHandler) write(writer http.ResponseWriter, status int, body []byte) {
	writer.WriteHeader(status)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	//if status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable {
	//	writer.Header().Set(httpResponseRetryAfter, handler.retryAfter)
	//}
	if body != nil {
		n := 0
		bodyLen := len(body)
		for n < bodyLen {
			nn, writeErr := writer.Write(body[n:])
			if writeErr != nil {
				return
			}
			n += nn
		}
	}
	return
}

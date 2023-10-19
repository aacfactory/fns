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

package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/caches"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/services/internal/secret"
	"github.com/aacfactory/fns/services/transports"
	transports2 "github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"golang.org/x/sync/singleflight"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	proxyHandlerName = "proxy"
)

func createProxyTransport(config *ProxyConfig, runtime *Runtime, registrations *Registrations,
	middlewares []TransportMiddleware, handlers []TransportHandler) (proxy *Transport, err error) {
	tr, registered := transports.Registered(strings.TrimSpace(config.Name))
	if !registered {
		err = errors.Warning("fns: create proxy failed").WithCause(errors.Warning("transport was not registered")).WithMeta("name", config.Name)
		return
	}
	midConfig, midConfigErr := config.MiddlewaresConfig()
	if midConfigErr != nil {
		err = errors.Warning("fns: create proxy failed").WithCause(midConfigErr)
		return
	}
	mid := newTransportMiddlewares(transportMiddlewaresOptions{
		Runtime: runtime,
		Cors:    config.Cors,
		Config:  midConfig,
	})
	if middlewares != nil && len(middlewares) > 0 {
		for _, middleware := range middlewares {
			appendErr := mid.Append(middleware)
			if appendErr != nil {
				err = errors.Warning("fns: create proxy failed").WithCause(appendErr)
				return
			}
		}
	}
	handlersConfig, handlersConfigErr := config.HandlersConfig()
	if handlersConfigErr != nil {
		err = errors.Warning("fns: create proxy failed").WithCause(handlersConfigErr)
		return
	}
	hds := newTransportHandlers(transportHandlersOptions{
		Runtime: runtime,
		Config:  handlersConfig,
	})

	if handlers == nil {
		handlers = make([]TransportHandler, 0, 1)
	}
	if config.EnableDevMode {
		handlers = append(handlers, newDevProxyHandler(registrations, runtime.signer))
	}

	handlers = append(handlers, newProxyHandler(proxyHandlerOptions{
		Signer:        runtime.Signature(),
		Registrations: registrations,
	}))

	for _, handler := range handlers {
		appendErr := hds.Append(handler)
		if appendErr != nil {
			err = errors.Warning("fns: create proxy failed").WithCause(appendErr)
			return
		}
	}

	options, optionsErr := config.ConvertToTransportsOptions(runtime.RootLog().With("fns", "proxy"), mid.Handler(hds))
	if optionsErr != nil {
		err = errors.Warning("fns: create proxy failed").WithCause(optionsErr)
		return
	}

	buildErr := tr.Build(options)
	if buildErr != nil {
		err = errors.Warning("fns: create proxy failed").WithCause(buildErr)
		return
	}

	port := options.Port
	proxy = &Transport{
		transport:   tr,
		middlewares: mid,
		handlers:    hds,
		port:        port,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type proxyHandlerOptions struct {
	Signer        *secret.Signer
	Registrations *Registrations
}

func newProxyHandler(options proxyHandlerOptions) (handler *proxyHandler) {
	handler = &proxyHandler{
		log:                    nil,
		disableHandleDocuments: false,
		disableHandleOpenapi:   false,
		openapiVersion:         "",
		appId:                  "",
		appName:                "",
		appVersion:             versions.Version{},
		signer:                 options.Signer,
		registrations:          options.Registrations,
		attachments:            caches.NewLRU[string, []byte](4),
		group:                  &singleflight.Group{},
	}
	return
}

type proxyHandlerConfig struct {
	DisableHandleDocuments bool   `json:"disableHandleDocuments"`
	DisableHandleOpenapi   bool   `json:"disableHandleOpenapi"`
	OpenapiVersion         string `json:"openapiVersion"`
	DocumentsTTL           string `json:"documentsTTL"`
}

type proxyHandler struct {
	log                    logs.Logger
	appId                  string
	appName                string
	appVersion             versions.Version
	disableHandleDocuments bool
	disableHandleOpenapi   bool
	openapiVersion         string
	documentsTTL           time.Duration
	signer                 *secret.Signer
	registrations          *Registrations
	attachments            *caches.LRU[string, []byte]
	group                  *singleflight.Group
}

func (handler *proxyHandler) Name() (name string) {
	name = proxyHandlerName
	return
}

func (handler *proxyHandler) Build(options TransportHandlerOptions) (err error) {
	config := proxyHandlerConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: build proxy handler failed").WithCause(configErr)
		return
	}
	handler.log = options.Log
	handler.appId = options.Runtime.AppId()
	handler.appName = options.Runtime.AppName()
	handler.appVersion = options.Runtime.AppVersion()
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
	return
}

func (handler *proxyHandler) Accept(r *transports2.Request) (ok bool) {
	ok = r.IsGet() && bytes.Compare(r.Path(), servicesDocumentsPath) == 0
	if ok {
		return
	}
	ok = r.IsGet() && bytes.Compare(r.Path(), servicesOpenapiPath) == 0
	if ok {
		return
	}
	ok = r.IsPost() && r.Header().Get(httpContentType) == httpContentTypeJson && r.Header().Get(httpRequestInternalSignatureHeader) == "" && len(r.PathResources()) == 2
	return
}

func (handler *proxyHandler) Close() (err error) {
	return
}

func (handler *proxyHandler) Handle(w transports2.ResponseWriter, r *transports2.Request) {
	if r.IsGet() && bytes.Compare(r.Path(), servicesOpenapiPath) == 0 {
		handler.handleOpenapi(w, r)
		return
	}
	if r.IsGet() && bytes.Compare(r.Path(), servicesDocumentsPath) == 0 {
		handler.handleDocuments(w, r)
		return
	}
	handler.handleProxy(w, r)
	return
}

func (handler *proxyHandler) fetchDocuments() (v documents.Documents, err error) {
	value, fetchErr, _ := handler.group.Do("documents:fetch", func() (v interface{}, err error) {
		doc, fetchErr := handler.registrations.ServiceDocuments()
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

func (handler *proxyHandler) handleOpenapi(w transports2.ResponseWriter, r *transports2.Request) {
	version := versions.Latest()
	r.Param("version")
	if targetVersion := bytex.ToString(r.Param("version")); targetVersion != "" {
		var err error
		version, err = versions.Parse(targetVersion)
		if err != nil {
			w.Failed(errors.Warning("proxy: parse version failed").WithCause(err))
			return
		}
	}

	key := fmt.Sprintf("openapi:%s", version.String())
	refresh := bytex.ToString(r.Param("refresh")) == "true"
	v, err, _ := handler.group.Do(fmt.Sprintf("%s:%v", key, refresh), func() (v interface{}, err error) {
		if !refresh {
			cached, has := handler.attachments.Get(key)
			if has {
				v = cached
				return
			}
		}
		var doc documents.Documents
		var docErr error
		cachedDoc, hasCachedDoc := handler.attachments.Get("documents")
		if hasCachedDoc {
			doc = documents.Documents{}
			docErr = json.Unmarshal(cachedDoc, &doc)
		} else {
			doc, docErr = handler.fetchDocuments()
		}
		if docErr != nil {
			err = errors.Warning("proxy: fetch backend documents failed").WithCause(docErr)
			return
		}
		api := doc.Openapi(handler.openapiVersion, handler.appId, handler.appName, version)
		p, encodeErr := json.Marshal(api)
		if encodeErr != nil {
			err = errors.Warning("proxy: encode openapi failed").WithCause(encodeErr)
			return
		}
		handler.attachments.Add(key, p, handler.documentsTTL)
		v = p
		return
	})
	if err != nil {
		w.Failed(errors.Map(err))
		return
	}
	handler.write(w, http.StatusOK, v.([]byte))
	return
}

func (handler *proxyHandler) handleDocuments(w transports2.ResponseWriter, r *transports2.Request) {
	key := "documents"
	refresh := bytex.ToString(r.Param("refresh")) == "true"
	v, err, _ := handler.group.Do(fmt.Sprintf("%s:write:%v", key, refresh), func() (v interface{}, err error) {
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
		v = p
		return
	})
	if err != nil {
		w.Failed(errors.Map(err))
		return
	}
	handler.write(w, http.StatusOK, v.([]byte))
	return
}

func (handler *proxyHandler) getDeviceId(r *transports2.Request) (devId string) {
	devId = strings.TrimSpace(r.Header().Get(httpDeviceIdHeader))
	return
}

func (handler *proxyHandler) getDeviceIp(r *transports2.Request) (devIp string) {
	devIp = r.Header().Get(httpDeviceIpHeader)
	return
}

func (handler *proxyHandler) getRequestId(r *transports2.Request) (requestId string, has bool) {
	requestId = strings.TrimSpace(r.Header().Get(httpRequestIdHeader))
	has = requestId != ""
	return
}

func (handler *proxyHandler) handleProxy(w transports2.ResponseWriter, r *transports2.Request) {
	// read path
	resources := r.PathResources()
	serviceNameBytes := resources[0]
	fnNameBytes := resources[1]
	serviceName := bytex.ToString(serviceNameBytes)
	fnName := bytex.ToString(fnNameBytes)
	// versions
	rvs, hasVersion, parseVersionErr := ParseRequestVersionFromHeader(r.Header())
	if parseVersionErr != nil {
		w.Failed(errors.Warning("fns: parse X-Fns-Request-Version failed").WithCause(parseVersionErr))
		return
	}
	if !hasVersion {
		rvs = AllowAllRequestVersions()
	}
	// read device
	deviceId := handler.getDeviceId(r)
	deviceIp := handler.getDeviceIp(r)
	// request id
	requestId, hasRequestId := handler.getRequestId(r)
	if !hasRequestId {
		requestId = uid.UID()
	}
	// timeout
	ctx := r.Context()
	var cancel context.CancelFunc
	timeout := r.Header().Get(httpRequestTimeoutHeader)
	if timeout != "" {
		timeoutMillisecond, parseTimeoutErr := strconv.ParseInt(timeout, 10, 64)
		if parseTimeoutErr != nil {
			w.Failed(errors.Warning("fns: X-Fns-Request-Timeout is not number").WithMeta("timeout", timeout))
			return
		}
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMillisecond)*time.Millisecond)
	}
	// discovery
	registration, has := handler.registrations.Get(serviceName, rvs)
	if !has {
		if cancel != nil {
			cancel()
		}
		w.Failed(errors.NotFound("fns: service was not found").WithMeta("service", serviceName))
		return
	}
	// request
	result, requestErr := registration.RequestSync(ctx, NewRequest(
		ctx,
		serviceName,
		fnName,
		NewArgument(r.Body()),
		WithRequestHeader(r.Header()),
		WithDeviceId(deviceId),
		WithDeviceIp(deviceIp),
		WithRequestId(requestId),
		WithRequestVersions(rvs),
	))
	if cancel != nil {
		cancel()
	}
	w.Header().Set(httpRequestIdHeader, requestId)
	if requestErr != nil {
		w.Failed(requestErr)
	} else {
		w.Succeed(result)
	}
	return
}

func (handler *proxyHandler) write(w transports2.ResponseWriter, status int, body []byte) {
	w.SetStatus(status)
	if body != nil {
		w.Header().Set(httpContentType, httpContentTypeJson)
		_, _ = w.Write(body)
	}
	return
}
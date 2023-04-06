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
	"github.com/aacfactory/fns/commons/caches"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/documents"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"golang.org/x/sync/singleflight"
	"net/http"
	"strings"
	"time"
)

const (
	proxyHandlerName = "proxy"
)

type proxyHandlerOptions struct {
	Signer        *secret.Signer
	Registrations *Registrations
	Dev           *devProxyHandler
	DeployedCh    <-chan map[string]*endpoint
}

// todo 只管正常代理和转发。开发由devProxy负责。且由于端口必然是services的端口，更加确凿的是，ssl如果存在，则必然是services的端口。
func newProxyHandler(options proxyHandlerOptions) (handler *proxyHandler) {
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
		registrations:          options.Registrations,
		attachments:            caches.NewLRU[string, []byte](4),
		group:                  &singleflight.Group{},
		dev:                    options.Dev,
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
	ready                  bool
	disableHandleDocuments bool
	disableHandleOpenapi   bool
	openapiVersion         string
	documentsTTL           time.Duration
	signer                 *secret.Signer
	registrations          *Registrations
	attachments            *caches.LRU[string, []byte]
	group                  *singleflight.Group
	dev                    *devProxyHandler
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
	return
}

func (handler *proxyHandler) Accept(r *transports.Request) (ok bool) {
	if handler.dev != nil {
		ok = handler.dev.Accept(r)
		if ok {
			r.Header().Set("dev", "true")
			return
		}
	}
	ok = r.IsGet() && bytex.ToString(r.Path()) == "/services/documents"
	if ok {
		return
	}
	ok = r.IsGet() && bytex.ToString(r.Path()) == "/services/openapi"
	if ok {
		return
	}
	ok = r.IsPost() && r.Header().Get(httpContentType) == httpContentTypeJson && len(strings.Split(bytex.ToString(r.Path()), "/")) == 3
	if ok {
		return
	}
	return
}

func (handler *proxyHandler) Close() {
	return
}

func (handler *proxyHandler) Handle(w transports.ResponseWriter, r *transports.Request) {
	if !handler.ready {
		w.Failed(ErrTooEarly.WithMeta("handler", handler.Name()))
		return
	}
	if r.Header().Get("dev") == "true" {
		r.Header().Del("dev")
		handler.dev.Handle(w, r)
		return
	}
	if r.IsGet() && bytex.ToString(r.Path()) == "/services/openapi" {
		handler.handleOpenapi(w, r)
		return
	}
	if r.IsGet() && bytex.ToString(r.Path()) == "/services/documents" {
		handler.handleDocuments(w, r)
		return
	}
	handler.handleProxy(w, r)
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

func (handler *proxyHandler) handleOpenapi(w transports.ResponseWriter, r *transports.Request) {
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

func (handler *proxyHandler) handleDocuments(w transports.ResponseWriter, r *transports.Request) {
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

func (handler *proxyHandler) handleProxy(w transports.ResponseWriter, r *transports.Request) {
	rvs, hasVersion, parseVersionErr := ParseRequestVersionFromHeader(r.Header())

	registration, has := handler.registrations.Get()
	registration.RequestSync()

	// TODO
	/*
			just as services, but use registrations insteadof discovery
			cause local was not served in proxy port
		see fasthttp
		func copyZeroAlloc(w io.Writer, r io.Reader) (int64, error) {
			vbuf := copyBufPool.Get()
			buf := vbuf.([]byte)
			n, err := io.CopyBuffer(w, r, buf)
			copyBufPool.Put(vbuf)
			return n, err
		}
	*/

	return
}

func (handler *proxyHandler) write(w transports.ResponseWriter, status int, body []byte) {
	w.SetStatus(status)
	if body != nil {
		w.Header().Set(httpContentType, httpContentTypeJson)
		_, _ = w.Write(body)
	}
	return
}

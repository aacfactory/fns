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
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/documents"
	"github.com/aacfactory/fns/service/internal/lru"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"golang.org/x/sync/singleflight"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type ServicesHandlerOptions struct {
	Signer         *secret.Signer
	DeployedCh     <-chan map[string]*endpoint
	OpenapiVersion string
	DevMode        bool
	Registrations  *Registrations
	Dialer         HttpClientDialer
}

func newServicesHandler(options ServicesHandlerOptions) (handler HttpHandler) {
	sh := &servicesHandler{
		log:              nil,
		ready:            false,
		names:            make([]string, 0, 1),
		documents:        documents.NewDocuments(),
		openapiVersion:   options.OpenapiVersion,
		proxyMode:        options.Registrations != nil,
		devMode:          options.DevMode,
		registrations:    options.Registrations,
		membersDocuments: lru.New[string, documents.Documents](64),
		dialer:           options.Dialer,
		appId:            "",
		appName:          "",
		appVersion:       versions.Version{},
		signer:           options.Signer,
		discovery:        nil,
		group:            &singleflight.Group{},
		cache:            nil,
	}
	go func(handler *servicesHandler, deployedCh <-chan map[string]*endpoint, openApiVersion string) {
		eps, ok := <-deployedCh
		handler.ready = true
		if !ok {
			return
		}
		if eps == nil || len(eps) == 0 {
			return
		}
		names := make([]string, 0, 1)
		for name, ep := range eps {
			handler.names = append(handler.names, name)
			names = append(names, name)
			if ep.Internal() || ep.Document() == nil {
				continue
			}
			handler.documents.Add(ep.Document())
		}
	}(sh, options.DeployedCh, options.OpenapiVersion)
	handler = sh
	return
}

type servicesHandlerConfig struct {
	Documents    string              `json:"documents"`
	Openapi      string              `json:"openapi"`
	CacheControl *CacheControlConfig `json:"cacheControl"`
}

type servicesHandler struct {
	log              logs.Logger
	ready            bool
	names            []string
	documents        documents.Documents
	openapiVersion   string
	proxyMode        bool
	devMode          bool
	registrations    *Registrations
	membersDocuments *lru.LRU[string, documents.Documents]
	dialer           HttpClientDialer
	appId            string
	appName          string
	appVersion       versions.Version
	signer           *secret.Signer
	discovery        EndpointDiscovery
	group            *singleflight.Group
	cache            *CacheControl
}

func (handler *servicesHandler) Name() (name string) {
	name = "services"
	return
}

func (handler *servicesHandler) Build(options *HttpHandlerOptions) (err error) {
	config := servicesHandlerConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: build services handler failed").WithCause(configErr)
		return
	}
	handler.log = options.Log.With("handler", "services")
	handler.appId = options.AppId
	handler.appName = options.AppName
	handler.appVersion = options.AppVersion
	handler.discovery = options.Discovery
	if config.CacheControl != nil {
		handler.cache, err = NewCacheControl(*config.CacheControl)
		if err != nil {
			err = errors.Warning("fns: build services handler failed").WithCause(err)
			return
		}
	}
	return
}

func (handler *servicesHandler) Accept(r *http.Request) (ok bool) {
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/documents"
	if ok {
		return
	}
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/openapi"
	if ok {
		return
	}
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/names"
	if ok {
		return
	}
	ok = r.Method == http.MethodPost && r.Header.Get(httpContentType) == httpContentTypeJson && len(strings.Split(r.URL.Path, "/")) == 3
	return
}

func (handler *servicesHandler) ServeHTTP(writer http.ResponseWriter, r *http.Request) {
	if !handler.ready {
		handler.failed(writer, errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: handler is not ready, try later again").WithMeta("handler", handler.Name()))
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/names" {
		if handler.proxyMode {
			handler.handleProxyNames(writer, r)
		} else {
			handler.handleNames(writer, r)
		}
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/documents" {
		if handler.proxyMode {
			handler.handleProxyDocuments(writer, r)
		} else {
			handler.handleDocuments(writer, r)
		}
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/openapi" {
		if handler.proxyMode {
			handler.handleProxyOpenapi(writer, r)
		} else {
			handler.handleOpenapi(writer, r)
		}
		return
	}
	// local internal
	if r.Header.Get(httpRequestInternalHeader) != "" {
		handler.handleInternalRequest(writer, r)
		return
	}
	// proxy
	if handler.proxyMode {
		if r.Header.Get(httpDevModeHeader) == "" {
			handler.handleProxyRequest(writer, r)
		} else {
			handler.handleDevProxyRequest(writer, r)
		}
		return
	}
	// local
	handler.handleRequest(writer, r)
	return
}

func (handler *servicesHandler) Close() {
	if handler.cache != nil {
		handler.cache.Close()
	}
	return
}

func (handler *servicesHandler) getDeviceIp(r *http.Request) (devIp string) {
	devIp = r.Header.Get(httpDeviceIpHeader)
	if devIp == "" {
		forwarded := r.Header.Get(httpXForwardedForHeader)
		if forwarded != "" {
			forwardedIps := strings.Split(forwarded, ",")
			devIp = strings.TrimSpace(forwardedIps[0])
		} else {
			remoteAddr := r.RemoteAddr
			if remoteAddr != "" {
				if strings.Index(remoteAddr, ".") > 0 && strings.Index(remoteAddr, ":") > 0 {
					devIp = remoteAddr[0:strings.Index(remoteAddr, ":")]
				}
			}
		}
	}
	return
}

func (handler *servicesHandler) handleRequest(writer http.ResponseWriter, r *http.Request) {
	pathItems := strings.Split(r.URL.Path, "/")
	if len(pathItems) != 3 {
		handler.failed(writer, errors.Warning("fns: invalid request url path"))
		return
	}
	if r.Header.Get(httpDeviceIdHeader) == "" {
		handler.failed(writer, errors.Warning("fns: X-Fns-Device-Id is required"))
		return
	}
	// read path
	serviceName := pathItems[1]
	fnName := pathItems[2]
	// check version
	rvs, hasVersion, parseVersionErr := ParseRequestVersionFromHeader(r.Header)
	if parseVersionErr != nil {
		handler.failed(writer, errors.Warning("fns: parse X-Fns-Request-Version failed").WithCause(parseVersionErr))
		return
	}
	if hasVersion && !rvs.Accept(serviceName, handler.appVersion) {
		handler.failed(writer,
			errors.Warning("fns: X-Fns-Request-Version was not matched").
				WithMeta("appVersion", handler.appVersion.String()).
				WithMeta("requestVersion", rvs.String()).
				WithMeta("service", serviceName).WithMeta("fn", fnName),
		)
		return
	} else {
		rvs = AllowAllRequestVersions()
	}
	// read body
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		handler.failed(writer, errors.Warning("fns: read body failed").WithCause(readBodyErr))
		return
	}
	_ = r.Body.Close()
	// cache control
	cacheKey := uint64(0)
	if handler.cache != nil {
		key, cached := handler.cache.Cached(r.Header, r.URL.Path, body)
		if cached {
			writer.WriteHeader(http.StatusNotModified)
			return
		}
		cacheKey = key
	}
	// devIp
	devIp := handler.getDeviceIp(r)
	// id
	id := uid.UID()
	// timeout
	ctx := r.Context()
	var cancel context.CancelFunc
	timeout := r.Header.Get(httpRequestTimeoutHeader)
	if timeout != "" {
		timeoutMillisecond, parseTimeoutErr := strconv.ParseInt(timeout, 10, 64)
		if parseTimeoutErr != nil {
			handler.failed(writer, errors.Warning("fns: X-Fns-Request-Timeout is not number").WithMeta("timeout", timeout))
			return
		}
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMillisecond)*time.Millisecond)
	}
	// discovery
	ep, hasEndpoint := handler.discovery.Get(ctx, serviceName, Native())
	if !hasEndpoint {
		if cancel != nil {
			cancel()
		}
		handler.failed(writer, errors.NotFound("fns: service was not found").WithMeta("service", serviceName))
		return
	}

	// request
	handleBegAT := time.Time{}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	responseHeader := http.Header{}
	result, requestErr := ep.RequestSync(withTracer(ctx, id), NewRequest(
		ctx,
		serviceName,
		fnName,
		NewArgument(body),
		WithHttpRequestHeader(r.Header),
		WithHttpResponseHeader(responseHeader),
		WithDeviceIp(devIp),
		WithRequestId(id),
		WithRequestVersions(rvs),
	))
	if cancel != nil {
		cancel()
	}
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	if requestErr != nil {
		handler.failed(writer, requestErr)
	} else {

		handler.succeed(writer, id, latency, cacheKey, responseHeader, result)
	}
	return
}

func (handler *servicesHandler) handleInternalRequest(writer http.ResponseWriter, r *http.Request) {
	if handler.cluster == nil {
		handler.failed(writer, errors.Warning("fns: cluster mode is not enabled"))
		return
	}
	// id
	id := r.Header.Get(httpRequestIdHeader)
	if id == "" {
		handler.failed(writer, errors.Warning("fns: no X-Fns-Request-Id in header"))
		return
	}
	pathItems := strings.Split(r.URL.Path, "/")
	serviceName := pathItems[1]
	fnName := pathItems[2]
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		handler.failed(writer, errors.Warning("fns: read body failed").WithCause(readBodyErr))
		return
	}
	_ = r.Body.Close()
	// verify signature
	if !handler.signer.Verify(body, bytex.FromString(r.Header.Get(httpRequestSignatureHeader))) {
		handler.failed(writer, errors.Warning("fns: signature is invalid"))
		return
	}
	// check version
	rvs, hasVersion, parseVersionErr := ParseRequestVersionFromHeader(r.Header)
	if parseVersionErr != nil {
		handler.failed(writer, errors.Warning("fns: parse X-Fns-Request-Version failed").WithCause(parseVersionErr))
		return
	}
	if hasVersion && !rvs.Accept(serviceName, handler.appVersion) {
		handler.failed(writer,
			errors.Warning("fns: X-Fns-Request-Version was not matched").
				WithMeta("appVersion", handler.appVersion.String()).
				WithMeta("requestVersion", rvs.String()),
		)
		return
	} else {
		rvs = AllowAllRequestVersions()
	}
	// cache control
	cacheKey := uint64(0)
	if handler.cache != nil {
		key, cached := handler.cache.Cached(r.Header, r.URL.Path, body)
		if cached {
			writer.WriteHeader(http.StatusNotModified)
			return
		}
		cacheKey = key
	}
	// internal request
	iReq := &internalRequestImpl{}
	decodeErr := json.Unmarshal(body, iReq)
	if decodeErr != nil {
		handler.failed(writer, errors.Warning("fns: decode body failed").WithCause(decodeErr))
		return
	}
	// timeout
	ctx := r.Context()
	var cancel context.CancelFunc
	timeout := r.Header.Get(httpRequestTimeoutHeader)
	if timeout != "" {
		timeoutMillisecond, parseTimeoutErr := strconv.ParseInt(timeout, 10, 64)
		if parseTimeoutErr != nil {
			handler.failed(writer, errors.Warning("fns: X-Fns-Request-Timeout is not number").WithMeta("timeout", timeout))
			return
		}
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMillisecond)*time.Millisecond)
	}

	// discovery
	ep, hasEndpoint := handler.discovery.Get(ctx, serviceName, Native())
	if !hasEndpoint {
		if cancel != nil {
			cancel()
		}
		handler.failed(writer, errors.NotFound("fns: service was not found").WithMeta("service", serviceName))
		return
	}

	// request
	responseHeader := http.Header{}
	req := NewRequest(
		ctx,
		serviceName,
		fnName,
		iReq.Argument,
		WithHttpRequestHeader(r.Header),
		WithHttpResponseHeader(responseHeader),
		WithRequestId(id),
		WithInternalRequest(),
		WithRequestTrunk(iReq.Trunk),
		WithRequestUser(iReq.User.Id(), iReq.User.Attributes()),
		WithRequestVersions(rvs),
	)

	result, requestErr := ep.RequestSync(withTracer(ctx, id), req)
	if cancel != nil {
		cancel()
	}

	var span *Span
	tracer_, hasTracer := GetTracer(ctx)
	if hasTracer {
		span = tracer_.RootSpan()
	}
	iResp := &internalResponse{
		User:  req.User(),
		Trunk: req.Trunk(),
		Span:  span,
		Body:  nil,
	}
	if requestErr != nil {
		iResp.Body = requestErr
		handler.writeInternalResponse(writer, requestErr.Code(), responseHeader, cacheKey, id, iResp)
	} else {
		if result.Exist() {
			iResp.Body = result
		}
		handler.writeInternalResponse(writer, http.StatusOK, responseHeader, cacheKey, id, iResp)
	}
	return
}

func (handler *servicesHandler) handleDocuments(writer http.ResponseWriter, r *http.Request) {
	deviceId := r.URL.Query().Get("deviceId")
	if deviceId == "" {
		handler.failed(writer, errors.Warning("fns: deviceId was required in query"))
		return
	}
	r.Header.Set(httpDeviceIdHeader, deviceId)

	v, err, _ := handler.group.Do(fmt.Sprintf("documents:%s", targetVersion.String()), func() (v interface{}, err error) {
		if r.URL.Query().Get("native") != "" && handler.cluster != nil {
			p, encodeErr := json.Marshal(handler.documents)
			if encodeErr != nil {
				handler.failed(writer, errors.Warning("fns: encode documents failed").WithCause(encodeErr))
				return
			}
			handler.write(writer, http.StatusOK, nil, p)
			return
		}

		return
	})

	handler.documents.Merge()
	if r.URL.Query().Get("native") != "" {

	} else {

	}
	// cache control
	cacheKey := uint64(0)
	if handler.cache != nil {
		r.Cookies()
		key, cached := handler.cache.Cached(r.Header, r.URL.String(), nil)
		if cached {
			writer.WriteHeader(http.StatusNotModified)
			return
		}
		cacheKey = key
	}

	writer.WriteHeader(http.StatusOK)
	writer.Header().Set(httpContentType, httpContentTypeJson)

	n := 0
	bodyLen := len(handler.documents)
	for n < bodyLen {
		nn, writeErr := writer.Write(handler.documents[n:])
		if writeErr != nil {
			return
		}
		n += nn
	}
	return
}

func (handler *servicesHandler) handleProxyDocuments(writer http.ResponseWriter, r *http.Request) {
	targetVersion := handler.appVersion
	version := r.URL.Query().Get("version")
	if version != "" {
		var parseVersionErr error
		targetVersion, parseVersionErr = versions.Parse(version)
		if parseVersionErr != nil {
			handler.failed(writer, errors.Warning("fns: version in query is invalid").WithCause(parseVersionErr))
			return
		}
	}

	return
}

func (handler *servicesHandler) handleOpenapi(writer http.ResponseWriter, r *http.Request) {
	writer.WriteHeader(http.StatusOK)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	n := 0
	bodyLen := len(handler.openapi)
	for n < bodyLen {
		nn, writeErr := writer.Write(handler.openapi[n:])
		if writeErr != nil {
			return
		}
		n += nn
	}
	return
}

func (handler *servicesHandler) handleProxyOpenapi(writer http.ResponseWriter, r *http.Request) {

	return
}

func (handler *servicesHandler) handleNames(writer http.ResponseWriter, r *http.Request) {
	deviceId := r.Header.Get(httpDeviceIdHeader)
	signature := r.Header.Get(httpRequestSignatureHeader)
	if !handler.signer.Verify([]byte(deviceId), []byte(signature)) {
		handler.failed(writer, errors.Warning("fns: invalid signature").WithMeta("handler", handler.Name()))
		return
	}
	writer.WriteHeader(http.StatusOK)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	n := 0
	bodyLen := len(handler.names)
	for n < bodyLen {
		nn, writeErr := writer.Write(handler.names[n:])
		if writeErr != nil {
			return
		}
		n += nn
	}
	return
}

func (handler *servicesHandler) handleProxyNames(writer http.ResponseWriter, r *http.Request) {

	return
}

func (handler *servicesHandler) handleProxyRequest(writer http.ResponseWriter, r *http.Request) {
	// todo set x-forwarded-for, when not exist
	return
}

func (handler *servicesHandler) handleDevProxyRequest(writer http.ResponseWriter, r *http.Request) {

	return
}

func (handler *servicesHandler) succeed(writer http.ResponseWriter, id string, latency time.Duration, cacheKey uint64, header http.Header, result interface{}) {
	body, encodeErr := json.Marshal(result)
	if encodeErr != nil {
		cause := errors.Warning("encode result failed").WithCause(encodeErr)
		handler.failed(writer, cause)
		return
	}
	writer.WriteHeader(http.StatusOK)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	// cache control
	if handler.cache != nil && cacheKey > 0 {
		age, hasAge := handler.cache.MaxAge(header)
		if hasAge {
			etag := handler.cache.CreateETag(cacheKey, age, body)
			writer.Header().Set(httpETagHeader, etag)
		}
	}
	// write header
	if header != nil && len(header) > 0 {
		for key, vv := range header {
			if vv == nil || len(vv) == 0 {
				continue
			}
			for _, v := range vv {
				writer.Header().Add(key, v)
			}
		}
	}
	if id != "" {
		writer.Header().Set(httpRequestIdHeader, id)
	}
	if handler.log.DebugEnabled() {
		writer.Header().Set(httpHandleLatencyHeader, latency.String())
	}
	// write body
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

func (handler *servicesHandler) failed(writer http.ResponseWriter, cause errors.CodeError) {
	if cause == nil {
		handler.write(writer, 555, nil, nil)
		return
	}
	status := cause.Code()
	if status == 0 {
		status = 555
	}
	body, _ := json.Marshal(cause)
	handler.write(writer, status, nil, body)
	return
}

func (handler *servicesHandler) writeInternalResponse(writer http.ResponseWriter, status int, header http.Header, cacheKey uint64, id string, data *internalResponse) {
	body, encodeErr := json.Marshal(data)
	if encodeErr != nil {
		data.Body = errors.Warning("fns: encode internal response failed").WithCause(encodeErr)
		body, _ = json.Marshal(data)
		status = 555
	}
	writer.WriteHeader(status)
	if status == http.StatusOK {
		// cache control
		if handler.cache != nil && cacheKey > 0 {
			age, hasAge := handler.cache.MaxAge(header)
			if hasAge {
				etag := handler.cache.CreateETag(cacheKey, age, body)
				writer.Header().Set(httpETagHeader, etag)
			}
		}
	}
	writer.Header().Set(httpContentType, httpContentTypeJson)
	if id != "" {
		writer.Header().Set(httpRequestIdHeader, id)
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

func (handler *servicesHandler) write(writer http.ResponseWriter, status int, header http.Header, body []byte) {
	writer.WriteHeader(status)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	if header != nil && len(header) > 0 {
		for k, vv := range header {
			if vv != nil && len(vv) > 0 {
				for _, v := range vv {
					writer.Header().Add(k, v)
				}
			}
		}
	}
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

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
	"github.com/aacfactory/fns/service/internal/ratelimit"
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
	Signer     *secret.Signer
	DeployedCh <-chan map[string]*endpoint
}

func newServicesHandler(options ServicesHandlerOptions) (handler HttpHandler) {
	sh := &servicesHandler{
		log:                    nil,
		ready:                  false,
		names:                  make([]string, 0, 1),
		documents:              documents.NewDocuments(),
		disableHandleDocuments: false,
		disableHandleOpenapi:   false,
		openapiVersion:         "",
		appId:                  "",
		appName:                "",
		appVersion:             versions.Version{},
		signer:                 options.Signer,
		discovery:              nil,
		group:                  &singleflight.Group{},
	}
	go func(handler *servicesHandler, deployedCh <-chan map[string]*endpoint) {
		eps, ok := <-deployedCh
		if !ok {
			return
		}
		handler.ready = true
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
	}(sh, options.DeployedCh)
	handler = sh
	return
}

type servicesHandlerConfig struct {
	DisableHandleDocuments bool                 `json:"disableHandleDocuments"`
	DisableHandleOpenapi   bool                 `json:"disableHandleOpenapi"`
	OpenapiVersion         string               `json:"openapiVersion"`
	Limiter                RequestLimiterConfig `json:"limiter"`
}

type servicesHandler struct {
	log                    logs.Logger
	ready                  bool
	names                  []string
	documents              documents.Documents
	disableHandleDocuments bool
	disableHandleOpenapi   bool
	openapiVersion         string
	appId                  string
	appName                string
	appVersion             versions.Version
	signer                 *secret.Signer
	discovery              EndpointDiscovery
	group                  *singleflight.Group
	limiter                *ratelimit.Limiter
	retryAfter             string
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
		handler.handleNames(writer, r)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/documents" {
		handler.handleDocuments(writer, r)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/openapi" {
		handler.handleOpenapi(writer, r)
		return
	}
	// internal
	if r.Header.Get(httpRequestInternalHeader) != "" {
		handler.handleInternalRequest(writer, r)
		return
	}
	handler.handleRequest(writer, r)
	return
}

func (handler *servicesHandler) Close() {
	return
}

func (handler *servicesHandler) getDeviceId(r *http.Request) (devId string, has bool) {
	devId = strings.TrimSpace(r.Header.Get(httpDeviceIdHeader))
	if devId == "" {
		devId = strings.TrimSpace(r.URL.Query().Get("deviceId"))
		if devId == "" {
			return
		}
		r.Header.Set(httpDeviceIdHeader, devId)
	}
	has = true
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
	// read device
	deviceId, hasDeviceId := handler.getDeviceId(r)
	if !hasDeviceId {
		handler.failed(writer, errors.Warning("fns: X-Fns-Device-Id was required in header"))
		return
	}
	deviceIp := handler.getDeviceIp(r)
	// limiter
	if !handler.limiter.Take(deviceId) {
		handler.failed(writer, ErrRequestOverload)
		return
	}
	defer handler.limiter.Repay(deviceId)
	// read path
	pathItems := strings.Split(r.URL.Path, "/")
	if len(pathItems) != 3 {
		handler.failed(writer, errors.Warning("fns: invalid request url path"))
		return
	}
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
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	result, requestErr := ep.RequestSync(withTracer(ctx, id), NewRequest(
		ctx,
		serviceName,
		fnName,
		NewArgument(body),
		WithHttpRequestHeader(r.Header),
		WithDeviceId(deviceId),
		WithDeviceIp(deviceIp),
		WithRequestId(id),
		WithRequestVersions(rvs),
	))
	if cancel != nil {
		cancel()
	}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	if requestErr != nil {
		handler.failed(writer, requestErr)
	} else {
		handler.succeed(writer, id, latency, result)
	}
	return
}

func (handler *servicesHandler) handleInternalRequest(writer http.ResponseWriter, r *http.Request) {
	// read device
	deviceId, hasDeviceId := handler.getDeviceId(r)
	if !hasDeviceId {
		handler.failed(writer, errors.Warning("fns: X-Fns-Device-Id was required in header"))
		return
	}
	deviceIp := handler.getDeviceIp(r)
	// limiter
	if !handler.limiter.Take(deviceId) {
		handler.failed(writer, ErrRequestOverload)
		return
	}
	defer handler.limiter.Repay(deviceId)
	// reade request id
	id := r.Header.Get(httpRequestIdHeader)
	if id == "" {
		handler.failed(writer, errors.Warning("fns: X-Fns-Request-Id was required in header"))
		return
	}
	// read path
	pathItems := strings.Split(r.URL.Path, "/")
	serviceName := pathItems[1]
	fnName := pathItems[2]
	// read body
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
	handleBegAT := time.Time{}
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	req := NewRequest(
		ctx,
		serviceName,
		fnName,
		iReq.Argument,
		WithHttpRequestHeader(r.Header),
		WithRequestId(id),
		WithDeviceId(deviceId),
		WithDeviceIp(deviceIp),
		WithInternalRequest(),
		WithRequestTrunk(iReq.Trunk),
		WithRequestUser(iReq.User.Id(), iReq.User.Attributes()),
		WithRequestVersions(rvs),
	)

	result, requestErr := ep.RequestSync(withTracer(ctx, id), req)
	if cancel != nil {
		cancel()
	}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	var span *Span
	tracer_, hasTracer := GetTracer(ctx)
	if hasTracer {
		span = tracer_.RootSpan()
	}
	resp := &internalResponse{
		User:  req.User(),
		Trunk: req.Trunk(),
		Span:  span,
		Body:  nil,
	}
	if requestErr == nil {
		resp.Succeed = true
		resp.Body = result
	} else {
		resp.Succeed = false
		resp.Body = requestErr
	}
	handler.succeed(writer, id, latency, resp)
	return
}

func (handler *servicesHandler) handleDocuments(writer http.ResponseWriter, r *http.Request) {
	if handler.disableHandleDocuments {
		handler.write(writer, http.StatusOK, bytex.FromString(emptyJson))
		return
	}
	const (
		key = "documents"
	)
	deviceId, hasDeviceId := handler.getDeviceId(r)
	if !hasDeviceId {
		handler.failed(writer, errors.Warning("fns: deviceId was required in query"))
		return
	}
	// limiter
	if !handler.limiter.Take(deviceId) {
		handler.failed(writer, ErrRequestOverload)
		return
	}
	defer handler.limiter.Repay(deviceId)
	// handle
	v, err, _ := handler.group.Do(key, func() (v interface{}, err error) {
		p, encodeErr := json.Marshal(handler.documents)
		if encodeErr != nil {
			handler.failed(writer, errors.Warning("fns: encode documents failed").WithCause(encodeErr))
			return
		}
		v = p
		return
	})
	if err != nil {
		handler.failed(writer, errors.Map(err))
		return
	}
	handler.write(writer, http.StatusOK, v.([]byte))
	return
}

func (handler *servicesHandler) handleOpenapi(writer http.ResponseWriter, r *http.Request) {
	if handler.disableHandleOpenapi {
		handler.write(writer, http.StatusOK, bytex.FromString(emptyJson))
		return
	}
	const (
		key = "openapi"
	)
	deviceId, hasDeviceId := handler.getDeviceId(r)
	if !hasDeviceId {
		handler.failed(writer, errors.Warning("fns: deviceId was required in query"))
		return
	}
	// limiter
	if !handler.limiter.Take(deviceId) {
		handler.failed(writer, ErrRequestOverload)
		return
	}
	defer handler.limiter.Repay(deviceId)
	// handle
	v, err, _ := handler.group.Do(key, func() (v interface{}, err error) {
		openapi := handler.documents.Openapi(handler.openapiVersion, handler.appId, handler.appName, handler.appVersion)
		p, encodeErr := json.Marshal(openapi)
		if encodeErr != nil {
			handler.failed(writer, errors.Warning("fns: encode openapi failed").WithCause(encodeErr))
			return
		}
		v = p
		return
	})
	if err != nil {
		handler.failed(writer, errors.Map(err))
		return
	}
	handler.write(writer, http.StatusOK, v.([]byte))
	return
}

func (handler *servicesHandler) handleNames(writer http.ResponseWriter, r *http.Request) {
	const (
		key = "names"
	)
	deviceId, hasDeviceId := handler.getDeviceId(r)
	if !hasDeviceId {
		handler.failed(writer, errors.Warning("fns: deviceId was required in query"))
		return
	}
	// limiter
	if !handler.limiter.Take(deviceId) {
		handler.failed(writer, ErrRequestOverload)
		return
	}
	defer handler.limiter.Repay(deviceId)
	// handle
	signature := r.Header.Get(httpRequestSignatureHeader)
	if !handler.signer.Verify([]byte(deviceId), []byte(signature)) {
		handler.failed(writer, errors.Warning("fns: invalid signature").WithMeta("handler", handler.Name()))
		return
	}
	v, err, _ := handler.group.Do(key, func() (v interface{}, err error) {
		p, encodeErr := json.Marshal(handler.names)
		if encodeErr != nil {
			handler.failed(writer, errors.Warning("fns: encode names failed").WithCause(encodeErr))
			return
		}
		v = p
		return
	})
	if err != nil {
		handler.failed(writer, errors.Map(err))
		return
	}
	handler.write(writer, http.StatusOK, v.([]byte))
	return
}

func (handler *servicesHandler) succeed(writer http.ResponseWriter, id string, latency time.Duration, result interface{}) {
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

func (handler *servicesHandler) failed(writer http.ResponseWriter, cause errors.CodeError) {
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

func (handler *servicesHandler) write(writer http.ResponseWriter, status int, body []byte) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	if status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable {
		writer.Header().Set(httpResponseRetryAfter, handler.retryAfter)
	}
	writer.WriteHeader(status)
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

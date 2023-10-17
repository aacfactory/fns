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
	"bytes"
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/documents"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/fns/service/transports"
	transports2 "github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"golang.org/x/sync/singleflight"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

var (
	servicesDocumentsPath = []byte("/services/documents")
	servicesOpenapiPath   = []byte("/services/openapi")
)

// +-------------------------------------------------------------------------------------------------------------------+

func createServiceTransport(config *TransportConfig, runtime *Runtime, middlewares []TransportMiddleware, handlers []TransportHandler) (tr *Transport, err error) {
	transport, registered := transports.Registered(strings.TrimSpace(config.Name))
	if !registered {
		err = errors.Warning("fns: create transport failed").WithCause(errors.Warning("transport was not registered")).WithMeta("name", config.Name)
		return
	}
	midConfig, midConfigErr := config.MiddlewaresConfig()
	if midConfigErr != nil {
		err = errors.Warning("fns: create transport failed").WithCause(midConfigErr)
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
				err = errors.Warning("fns: create transport failed").WithCause(appendErr)
				return
			}
		}
	}
	handlersConfig, handlersConfigErr := config.HandlersConfig()
	if handlersConfigErr != nil {
		err = errors.Warning("fns: create transport failed").WithCause(handlersConfigErr)
		return
	}
	hds := newTransportHandlers(transportHandlersOptions{
		Runtime: runtime,
		Config:  handlersConfig,
	})
	if handlers == nil {
		handlers = make([]TransportHandler, 0, 1)
	}
	handlers = append(handlers, newServicesHandler(servicesHandlerOptions{
		Signer: runtime.Signature(),
	}))
	for _, handler := range handlers {
		appendErr := hds.Append(handler)
		if appendErr != nil {
			err = errors.Warning("fns: create transport failed").WithCause(appendErr)
			return
		}
	}

	options, optionsErr := config.ConvertToTransportsOptions(runtime.RootLog().With("fns", "transport"), mid.Handler(hds))
	if optionsErr != nil {
		err = errors.Warning("fns: create transport failed").WithCause(optionsErr)
		return
	}

	buildErr := transport.Build(options)
	if buildErr != nil {
		err = errors.Warning("fns: create transport failed").WithCause(buildErr)
		return
	}
	port := options.Port
	tr = &Transport{
		transport:   transport,
		middlewares: mid,
		handlers:    hds,
		port:        port,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type servicesHandlerOptions struct {
	Signer *secret.Signer
}

func newServicesHandler(options servicesHandlerOptions) (handler TransportHandler) {
	handler = &servicesHandler{
		log:                    nil,
		documents:              nil,
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
	return
}

type servicesHandlerConfig struct {
	DisableHandleDocuments bool   `json:"disableHandleDocuments"`
	DisableHandleOpenapi   bool   `json:"disableHandleOpenapi"`
	OpenapiVersion         string `json:"openapiVersion"`
}

type servicesHandler struct {
	log                    logs.Logger
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
}

func (handler *servicesHandler) Name() (name string) {
	name = "services"
	return
}

func (handler *servicesHandler) Build(options TransportHandlerOptions) (err error) {
	config := servicesHandlerConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: build services handler failed").WithCause(configErr)
		return
	}
	handler.log = options.Log.With("handler", "services")
	handler.appId = options.Runtime.AppId()
	handler.appName = options.Runtime.AppName()
	handler.appVersion = options.Runtime.AppVersion()
	handler.discovery = options.Runtime.discovery
	handler.disableHandleDocuments = config.DisableHandleDocuments
	handler.disableHandleOpenapi = config.DisableHandleOpenapi
	if !handler.disableHandleOpenapi {
		handler.openapiVersion = strings.TrimSpace(config.OpenapiVersion)
	}
	return
}

func (handler *servicesHandler) Accept(r *transports2.Request) (ok bool) {
	if ok = r.IsGet() && bytes.Compare(r.Path(), servicesDocumentsPath) == 0; ok {
		return
	}
	if ok = r.IsGet() && bytes.Compare(r.Path(), servicesOpenapiPath) == 0; ok {
		return
	}
	if ok = r.IsPost() && r.Header().Get(httpContentType) == httpContentTypeJson && len(r.PathResources()) == 2; ok {
		return
	}
	return
}

func (handler *servicesHandler) Handle(w transports2.ResponseWriter, r *transports2.Request) {
	if r.IsGet() && bytes.Compare(r.Path(), servicesDocumentsPath) == 0 {
		handler.handleDocuments(w, r)
		return
	}
	if r.IsGet() && bytes.Compare(r.Path(), servicesOpenapiPath) == 0 {
		handler.handleOpenapi(w, r)
		return
	}
	// internal
	if r.Header().Get(httpRequestInternalHeader) != "" {
		handler.handleInternalRequest(w, r)
		return
	}
	handler.handleRequest(w, r)
	return
}

func (handler *servicesHandler) Close() (err error) {
	return
}

func (handler *servicesHandler) getDeviceId(r *transports2.Request) (devId string) {
	devId = strings.TrimSpace(r.Header().Get(httpDeviceIdHeader))
	return
}

func (handler *servicesHandler) getDeviceIp(r *transports2.Request) (devIp string) {
	devIp = r.Header().Get(httpDeviceIpHeader)
	return
}

func (handler *servicesHandler) getRequestId(r *transports2.Request) (requestId string, has bool) {
	requestId = strings.TrimSpace(r.Header().Get(httpRequestIdHeader))
	has = requestId != ""
	return
}

func (handler *servicesHandler) handleRequest(writer transports2.ResponseWriter, r *transports2.Request) {
	// read path
	resources := r.PathResources()
	serviceNameBytes := resources[0]
	fnNameBytes := resources[1]
	serviceName := bytex.ToString(serviceNameBytes)
	fnName := bytex.ToString(fnNameBytes)
	// check version
	rvs, hasVersion, parseVersionErr := ParseRequestVersionFromHeader(r.Header())
	if parseVersionErr != nil {
		handler.failed(writer, "", errors.Warning("fns: parse X-Fns-Request-Version failed").WithCause(parseVersionErr))
		return
	}
	if hasVersion && !rvs.Accept(serviceName, handler.appVersion) {
		handler.failed(writer, "",
			errors.Warning("fns: X-Fns-Request-Version was not matched").
				WithMeta("appVersion", handler.appVersion.String()).
				WithMeta("requestVersion", rvs.String()).
				WithMeta("service", serviceName).WithMeta("fn", fnName),
		)
		return
	} else {
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
			handler.failed(writer, requestId, errors.Warning("fns: X-Fns-Request-Timeout is not number").WithMeta("timeout", timeout))
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
		handler.failed(writer, requestId, errors.NotFound("fns: service was not found").WithMeta("service", serviceName))
		return
	}

	// request
	result, requestErr := ep.RequestSync(withTracer(ctx, requestId), NewRequest(
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
	if requestErr != nil {
		handler.failed(writer, requestId, requestErr)
	} else {
		handler.succeed(writer, requestId, result)
	}
	return
}

func (handler *servicesHandler) handleInternalRequest(writer transports2.ResponseWriter, r *transports2.Request) {
	// reade request id
	requestId, hasRequestId := handler.getRequestId(r)
	if !hasRequestId {
		handler.failed(writer, requestId, errors.Warning("fns: X-Fns-Request-Id was required in header"))
		return
	}

	// read path
	pathItems := strings.Split(bytex.ToString(r.Path()), "/")
	serviceName := pathItems[1]
	fnName := pathItems[2]
	// read body
	body := r.Body()
	// verify signature
	if !handler.signer.Verify(body, bytex.FromString(r.Header().Get(httpRequestInternalSignatureHeader))) {
		handler.failed(writer, requestId, errors.Warning("fns: signature is invalid"))
		return
	}
	// check version
	rvs, hasVersion, parseVersionErr := ParseRequestVersionFromHeader(r.Header())
	if parseVersionErr != nil {
		handler.failed(writer, requestId, errors.Warning("fns: parse X-Fns-Request-Version failed").WithCause(parseVersionErr))
		return
	}
	if hasVersion && !rvs.Accept(serviceName, handler.appVersion) {
		handler.failed(writer,
			requestId,
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
		handler.failed(writer, requestId, errors.Warning("fns: decode body failed").WithCause(decodeErr))
		return
	}
	// timeout
	ctx := r.Context()
	var cancel context.CancelFunc
	timeout := r.Header().Get(httpRequestTimeoutHeader)
	if timeout != "" {
		timeoutMillisecond, parseTimeoutErr := strconv.ParseInt(timeout, 10, 64)
		if parseTimeoutErr != nil {
			handler.failed(writer, requestId, errors.Warning("fns: X-Fns-Request-Timeout is not number").WithMeta("timeout", timeout))
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
		handler.failed(writer, requestId, errors.NotFound("fns: service was not found").WithMeta("service", serviceName))
		return
	}
	// read device
	deviceId := handler.getDeviceId(r)
	deviceIp := handler.getDeviceIp(r)

	// request
	req := NewRequest(
		ctx,
		serviceName,
		fnName,
		iReq.Argument,
		WithRequestHeader(r.Header()),
		WithRequestId(requestId),
		WithDeviceId(deviceId),
		WithDeviceIp(deviceIp),
		WithInternalRequest(),
		WithRequestTrunk(iReq.Trunk),
		WithRequestUser(iReq.User.Id(), iReq.User.Attributes()),
		WithRequestVersions(rvs),
	)

	result, requestErr := ep.RequestSync(withTracer(ctx, requestId), req)
	if cancel != nil {
		cancel()
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
	handler.succeed(writer, requestId, resp)
	return
}

func (handler *servicesHandler) createDocuments(r *transports2.Request) {
	handler.documents = documents.NewDocuments()
	rt := GetRuntime(r.Context())
	namePlates := rt.ServiceNames()
	if len(namePlates) > 0 {
		for _, namePlate := range namePlates {
			if !namePlate.Internal() && namePlate.Document() != nil {
				handler.documents.Add(namePlate.Document())
			}
		}
	}
}

func (handler *servicesHandler) handleDocuments(w transports2.ResponseWriter, r *transports2.Request) {
	if handler.disableHandleDocuments {
		w.Failed(errors.Warning("fns: documents handler was disabled"))
		return
	}
	const (
		key = "documents"
	)
	// handle
	v, err, _ := handler.group.Do(key, func() (v interface{}, err error) {
		if handler.documents == nil {
			handler.createDocuments(r)
		}
		if handler.documents.Len() > 0 {
			v = []byte{'{', '}'}
			return
		}
		p, encodeErr := json.Marshal(handler.documents)
		if encodeErr != nil {
			err = errors.Warning("fns: encode documents failed").WithCause(encodeErr)
			return
		}
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

func (handler *servicesHandler) handleOpenapi(w transports2.ResponseWriter, r *transports2.Request) {
	if handler.disableHandleOpenapi {
		w.Succeed(Empty{})
		return
	}
	const (
		key = "openapi"
	)
	// handle
	v, err, _ := handler.group.Do(key, func() (v interface{}, err error) {
		if handler.documents == nil {
			handler.createDocuments(r)
		}
		openapi := handler.documents.Openapi(handler.openapiVersion, handler.appId, handler.appName, handler.appVersion)
		p, encodeErr := json.Marshal(openapi)
		if encodeErr != nil {
			err = errors.Warning("fns: encode openapi failed").WithCause(encodeErr)
			return
		}
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

func (handler *servicesHandler) succeed(w transports2.ResponseWriter, id string, result interface{}) {
	if id != "" {
		w.Header().Set(httpRequestIdHeader, id)
	}
	w.Succeed(result)
	return
}

func (handler *servicesHandler) failed(w transports2.ResponseWriter, id string, cause errors.CodeError) {
	if id != "" {
		w.Header().Set(httpRequestIdHeader, id)
	}
	w.Failed(cause)
	return
}

func (handler *servicesHandler) write(w transports2.ResponseWriter, status int, body []byte) {
	w.SetStatus(status)
	if body != nil {
		w.Header().Set(httpContentType, httpContentTypeJson)
		_, _ = w.Write(body)
	}
	return
}

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
	stdjson "encoding/json"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

func newServiceHandler(secretKey []byte, internalRequestEnabled bool, deployedCh chan map[string]*endpoint, openApiVersion string) (handler HttpHandler) {
	sh := &servicesHandler{
		log:                    nil,
		names:                  []byte{'[', ']'},
		namesWithInternal:      []byte{'[', ']'},
		documents:              []byte{'{', '}'},
		openapi:                []byte{'{', '}'},
		appId:                  "",
		appName:                "",
		appVersion:             versions.Version{},
		internalRequestEnabled: internalRequestEnabled,
		signer:                 secret.NewSigner(secretKey),
		discovery:              nil,
	}
	go func(handler *servicesHandler, deployedCh chan map[string]*endpoint, openApiVersion string) {
		eps, ok := <-deployedCh
		if !ok {
			return
		}
		if eps == nil || len(eps) == 0 {
			return
		}
		names := make([]string, 0, 1)
		namesWithInternal := make([]string, 0, 1)
		documents := make(map[string]Document)
		for name, ep := range eps {
			namesWithInternal = append(namesWithInternal, name)
			if !ep.Internal() {
				names = append(names, name)
				document := ep.Document()
				if document != nil {
					documents[name] = document
				}
			}
		}
		namesBytes, namesErr := json.Marshal(names)
		if namesErr == nil {
			handler.names = namesBytes
		}
		namesWithInternalBytes, namesWithInternalErr := json.Marshal(namesWithInternal)
		if namesWithInternalErr == nil {
			handler.namesWithInternal = namesWithInternalBytes
		}
		document, documentErr := encodeDocuments(handler.appId, handler.appName, handler.appVersion, eps)
		if documentErr == nil {
			handler.documents = document
		}
		openapi, openapiErr := encodeOpenapi(openApiVersion, handler.appId, handler.appName, handler.appVersion, eps)
		if openapiErr == nil {
			handler.openapi = openapi
		}
	}(sh, deployedCh, openApiVersion)
	handler = sh
	return
}

type servicesHandler struct {
	log                    logs.Logger
	names                  []byte
	namesWithInternal      []byte
	documents              []byte
	openapi                []byte
	appId                  string
	appName                string
	appVersion             versions.Version
	internalRequestEnabled bool
	signer                 *secret.Signer
	discovery              EndpointDiscovery
}

func (handler *servicesHandler) Name() (name string) {
	name = "services"
	return
}

func (handler *servicesHandler) Build(options *HttpHandlerOptions) (err error) {
	handler.log = options.Log.With("handler", "services")
	handler.appId = options.AppId
	handler.appName = options.AppName
	handler.appVersion = options.AppVersion
	handler.discovery = options.Discovery
	return
}

func (handler *servicesHandler) Accept(r *http.Request) (ok bool) {
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/names"
	if ok {
		return
	}
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/documents"
	if ok {
		return
	}
	ok = r.Method == http.MethodGet && r.URL.Path == "/services/openapi"
	if ok {
		return
	}
	ok = r.Method == http.MethodPost && r.Header.Get(httpContentType) == httpContentTypeJson && len(strings.Split(r.URL.Path, "/")) == 3
	return
}

func (handler *servicesHandler) ServeHTTP(writer http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/services/names" {
		handler.handleNames(writer, r)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/documents" {
		handler.handleDocuments(writer)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/openapi" {
		handler.handleOpenapi(writer)
		return
	}
	handler.handleRequest(writer, r)
	return
}

func (handler *servicesHandler) Close() {
	return
}

func (handler *servicesHandler) matchRequestVersion(writer http.ResponseWriter, r *http.Request) (ok bool) {
	version := r.Header.Get(httpRequestVersionsHeader)
	if version != "" {
		leftVersion := versions.Version{}
		rightVersion := versions.Version{}
		var parseVersionErr error
		versionRange := strings.Split(version, ",")
		leftVersionValue := strings.TrimSpace(versionRange[0])
		if leftVersionValue != "" {
			leftVersion, parseVersionErr = versions.Parse(leftVersionValue)
			if parseVersionErr != nil {
				handler.failed(writer, "", 0, http.StatusNotAcceptable, errors.NotAcceptable("fns: read request version failed").WithCause(parseVersionErr))
				return
			}
		}
		if len(versionRange) > 1 {
			rightVersionValue := strings.TrimSpace(versionRange[1])
			if rightVersionValue != "" {
				rightVersion, parseVersionErr = versions.Parse(rightVersionValue)
				if parseVersionErr != nil {
					handler.failed(writer, "", 0, http.StatusNotAcceptable, errors.NotAcceptable("fns: read request version failed").WithCause(parseVersionErr))
					return
				}
			}
		}
		if !handler.appVersion.Between(leftVersion, rightVersion) {
			handler.failed(writer, "", 0, http.StatusNotAcceptable, errors.NotAcceptable("fns: request version is not acceptable").WithMeta("version", handler.appVersion.String()))
			return
		}
	}
	ok = true
	return
}

func (handler *servicesHandler) getDeviceIp(r *http.Request) (devIp string) {
	devIp = r.Header.Get(httpDeviceIpHeader)
	if devIp == "" {
		forwarded := r.Header.Get(httpXForwardedForHeader)
		if forwarded != "" {
			forwardedIps := strings.Split(forwarded, ",")
			devIp = strings.TrimSpace(forwardedIps[len(forwardedIps)-1])
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
		handler.failed(writer, "", 0, http.StatusBadRequest, errors.BadRequest("fns: invalid request url path"))
		return
	}
	if r.Header.Get(httpDeviceIdHeader) == "" {
		handler.failed(writer, "", 0, http.StatusBadRequest, errors.BadRequest("fns: X-Fns-Device-Id is required"))
		return
	}

	if r.Header.Get(httpRequestInternalHeader) != "" {
		handler.handleInternalRequest(writer, r)
		return
	}

	serviceName := pathItems[1]
	fnName := pathItems[2]
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		handler.failed(writer, "", 0, http.StatusBadRequest, errors.BadRequest("fns: read body failed").WithCause(readBodyErr))
		return
	}

	if !handler.matchRequestVersion(writer, r) {
		return
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
			handler.failed(writer, "", 0, http.StatusBadRequest, errors.BadRequest("fns: X-Fns-Request-Timeout is not number").WithMeta("timeout", timeout))
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
		handler.failed(writer, "", 0, http.StatusNotFound, errors.NotFound("fns: service was not found").WithMeta("service", serviceName))
		return
	}

	// request
	handleBegAT := time.Time{}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	result, hasResult, requestErr := ep.RequestSync(withTracer(ctx, id), NewRequest(
		ctx,
		serviceName,
		fnName,
		NewArgument(body),
		WithHttpRequestHeader(r.Header),
		WithDeviceIp(devIp),
		WithRequestId(id),
	))
	if cancel != nil {
		cancel()
	}
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	if requestErr != nil {
		handler.failed(writer, id, latency, requestErr.Code(), requestErr)
	} else {
		if hasResult {
			switch result.(type) {
			case []byte:
				handler.succeed(writer, id, latency, result.([]byte))
				break
			case json.RawMessage:
				handler.succeed(writer, id, latency, result.(json.RawMessage))
				break
			case stdjson.RawMessage:
				handler.succeed(writer, id, latency, result.(stdjson.RawMessage))
				break
			default:
				p, encodeErr := json.Marshal(result)
				if encodeErr != nil {
					handler.failed(writer, id, latency, 555, errors.Warning("fns: encoding result failed").WithCause(encodeErr))
				} else {
					handler.succeed(writer, id, latency, p)
				}
				break
			}
		} else {
			handler.succeed(writer, id, latency, []byte{'{', '}'})
		}
	}
	return
}

func (handler *servicesHandler) handleInternalRequest(writer http.ResponseWriter, r *http.Request) {
	if !handler.internalRequestEnabled {
		handler.failed(writer, "", 0, http.StatusNotAcceptable, errors.NotAcceptable("fns: cluster mode is disabled"))
		return
	}
	// id
	id := r.Header.Get(httpRequestIdHeader)
	if id == "" {
		handler.failed(writer, "", 0, http.StatusNotAcceptable, errors.NotAcceptable("fns: no X-Fns-Request-Id in header"))
		return
	}
	pathItems := strings.Split(r.URL.Path, "/")
	serviceName := pathItems[1]
	fnName := pathItems[2]
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		handler.failed(writer, "", 0, http.StatusBadRequest, errors.BadRequest("fns: read body failed").WithCause(readBodyErr))
		return
	}
	// verify signature
	if !handler.signer.Verify(body, bytex.FromString(r.Header.Get(httpRequestSignatureHeader))) {
		handler.failed(writer, "", 0, http.StatusNotAcceptable, errors.NotAcceptable("fns: signature is invalid"))
		return
	}
	if !handler.matchRequestVersion(writer, r) {
		return
	}
	// internal request
	iReq := &internalRequestImpl{}
	decodeErr := json.Unmarshal(body, iReq)
	if decodeErr != nil {
		handler.failed(writer, "", 0, http.StatusNotAcceptable, errors.NotAcceptable("fns: decode body failed").WithCause(decodeErr))
		return
	}
	// timeout
	ctx := r.Context()
	var cancel context.CancelFunc
	timeout := r.Header.Get(httpRequestTimeoutHeader)
	if timeout != "" {
		timeoutMillisecond, parseTimeoutErr := strconv.ParseInt(timeout, 10, 64)
		if parseTimeoutErr != nil {
			handler.failed(writer, "", 0, http.StatusNotAcceptable, errors.BadRequest("fns: X-Fns-Request-Timeout is not number").WithMeta("timeout", timeout))
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
		handler.failed(writer, "", 0, http.StatusNotFound, errors.NotFound("fns: service was not found").WithMeta("service", serviceName))
		return
	}

	// request
	req := NewRequest(
		ctx,
		serviceName,
		fnName,
		NewArgument(iReq.Body),
		WithHttpRequestHeader(r.Header),
		WithRequestId(id),
		WithInternalRequest(),
		WithRequestTrunk(iReq.Trunk),
		WithRequestUser(iReq.User.Id(), iReq.User.Attributes()),
	)

	result, hasResult, requestErr := ep.RequestSync(withTracer(ctx, id), req)
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
		requestErrBytes, encodeErr := json.Marshal(requestErr)
		if encodeErr != nil {
			if handler.log.WarnEnabled() {
				handler.log.Warn().Cause(encodeErr).Message("fns: service handle encode request error failed")
			}
			requestErrBytes = bytex.FromString(`{"message":"` + requestErr.Message() + `"}`)
		}
		iResp.Body = requestErrBytes
		handler.failed(writer, id, 0, requestErr.Code(), iResp)
	} else {
		if hasResult {
			switch result.(type) {
			case []byte:
				resultBytes := result.([]byte)
				if json.Validate(resultBytes) {
					iResp.Body = resultBytes
				} else {
					resultJsonBytes, encodeErr := json.Marshal(result)
					if encodeErr != nil {
						if handler.log.WarnEnabled() {
							handler.log.Warn().Cause(encodeErr).Message("fns: service handle encode request error failed")
						}
						resultJsonBytes, _ = json.Marshal(errors.Warning("fns: service handler encode internal result failed").WithCause(encodeErr))
						iResp.Body = resultJsonBytes
						handler.failed(writer, id, 0, 555, iResp)
						return
					}
					iResp.Body = resultJsonBytes
				}
				break
			case json.RawMessage:
				iResp.Body = result.(json.RawMessage)
				break
			case stdjson.RawMessage:
				iResp.Body = json.RawMessage(result.(stdjson.RawMessage))
				break
			default:
				resultJsonBytes, encodeErr := json.Marshal(result)
				if encodeErr != nil {
					if handler.log.WarnEnabled() {
						handler.log.Warn().Cause(encodeErr).Message("fns: service handle encode request error failed")
					}
					resultJsonBytes, _ = json.Marshal(errors.Warning("fns: service handler encode internal result failed").WithCause(encodeErr))
					iResp.Body = resultJsonBytes
					handler.failed(writer, id, 0, 555, iResp)
					return
				}
				iResp.Body = resultJsonBytes
				break
			}
		} else {
			iResp.Body = []byte{'{', '}'}
		}
		handler.succeed(writer, id, 0, iResp)
	}

	return
}

func (handler *servicesHandler) handleDocuments(writer http.ResponseWriter) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(http.StatusOK)
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

func (handler *servicesHandler) handleOpenapi(writer http.ResponseWriter) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(http.StatusOK)
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

func (handler *servicesHandler) handleNames(writer http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get(httpRequestSignatureHeader)
	withInternal := false
	if signature != "" {
		deviceId := r.Header.Get(httpDeviceIdHeader)
		if deviceId != "" {
			withInternal = handler.signer.Verify([]byte(deviceId), []byte(signature))
		}
	}
	if withInternal {
		handler.succeed(writer, "", 0, handler.namesWithInternal)
	} else {
		handler.succeed(writer, "", 0, handler.names)
	}
	return
}

func (handler *servicesHandler) succeed(writer http.ResponseWriter, id string, latency time.Duration, result interface{}) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	if id != "" {
		writer.Header().Set(httpRequestIdHeader, id)
	}
	if handler.log.DebugEnabled() {
		writer.Header().Set(httpHandleLatencyHeader, latency.String())
	}
	writer.WriteHeader(http.StatusOK)
	if result == nil {
		_, _ = writer.Write([]byte{'{', '}'})
		return
	}
	body, encodeErr := json.Marshal(result)
	if encodeErr != nil {
		cause := errors.ServiceError("encode result failed").WithCause(encodeErr)
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

func (handler *servicesHandler) failed(writer http.ResponseWriter, id string, latency time.Duration, status int, cause interface{}) {
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

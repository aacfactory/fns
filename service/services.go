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

func newServiceHandler(appId string, appName string, appVersion versions.Version, log logs.Logger, internalRequestEnabled bool, signer *secret.Signer, endpoints map[string]Endpoint) (handler HttpHandler) {
	names := make([]string, 0, 1)
	namesWithInternal := make([]string, 0, 1)
	for name, ep := range endpoints {
		namesWithInternal = append(namesWithInternal, name)
		if ep.Internal() {
			continue
		}
		names = append(names, name)
	}
	np, _ := json.Marshal(names)
	nip, _ := json.Marshal(namesWithInternal)
	dp, dpErr := encodeDocuments(appId, appName, appVersion, endpoints)
	if dpErr != nil {
		if log.DebugEnabled() {
			log.Debug().Cause(dpErr).Message("fns: create services handler failed")
		}
		dp = []byte("{}")
	}
	op, opErr := encodeOpenapi(appId, appName, appVersion, endpoints)
	if opErr != nil {
		if log.DebugEnabled() {
			log.Debug().Cause(opErr).Message("fns: create services handler failed")
		}
		dp = []byte("{}")
	}
	handler = &servicesHandler{
		log:                    log.With("handler", "services"),
		names:                  np,
		namesWithInternal:      nip,
		documents:              dp,
		openapi:                op,
		appVersion:             appVersion,
		signer:                 signer,
		internalRequestEnabled: internalRequestEnabled,
		endpoints:              endpoints,
	}
	return
}

type servicesHandler struct {
	log                    logs.Logger
	names                  []byte
	namesWithInternal      []byte
	documents              []byte
	openapi                []byte
	appVersion             versions.Version
	signer                 *secret.Signer
	internalRequestEnabled bool
	endpoints              map[string]Endpoint
}

func (handler *servicesHandler) Name() (name string) {
	name = "services"
	return
}

func (handler *servicesHandler) Build(_ *HttpHandlerOptions) (err error) {
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
		handler.handleDocuments(writer, r)
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/services/openapi" {
		handler.handleOpenapi(writer, r)
		return
	}
	handler.handleRequest(writer, r)
	return
}

func (handler *servicesHandler) Close() {
	return
}

func (handler *servicesHandler) handleRequest(writer http.ResponseWriter, r *http.Request) {
	pathItems := strings.Split(r.URL.Path, "/")
	if len(pathItems) != 3 {
		handler.failed(writer, "", 0, errors.BadRequest("fns: invalid request url path"))
		return
	}
	devId := r.Header.Get(httpDeviceIdHeader)
	if devId == "" {
		handler.failed(writer, "", 0, errors.BadRequest("fns: X-Fns-Device-Id is required"))
		return
	}
	serviceName := pathItems[1]
	fnName := pathItems[2]
	body, readBodyErr := io.ReadAll(r.Body)
	if readBodyErr != nil {
		handler.failed(writer, "", 0, errors.BadRequest("fns: read body failed").WithCause(readBodyErr))
		return
	}

	// sign
	signed := false
	sign := r.Header.Get(httpRequestSignatureHeader)
	if sign != "" {
		if !handler.signer.Verify(body, bytex.FromString(sign)) {
			handler.failed(writer, "", 0, errors.NotAcceptable("fns: signature is invalid"))
			return
		}
		signed = true
	}

	// internal
	internal := false
	if r.Header.Get(httpRequestInternalHeader) != "" {
		if !handler.internalRequestEnabled {
			handler.failed(writer, "", 0, errors.NotAcceptable("fns: cluster mode is disabled"))
			return
		}
		if signed {
			internal = true
		}
	}

	// version
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
				handler.failed(writer, "", 0, errors.NotAcceptable("fns: read request version failed").WithCause(parseVersionErr))
				return
			}
		}
		if len(versionRange) > 1 {
			rightVersionValue := strings.TrimSpace(versionRange[1])
			if rightVersionValue != "" {
				rightVersion, parseVersionErr = versions.Parse(rightVersionValue)
				if parseVersionErr != nil {
					handler.failed(writer, "", 0, errors.NotAcceptable("fns: read request version failed").WithCause(parseVersionErr))
					return
				}
			}
		}
		if !handler.appVersion.Between(leftVersion, rightVersion) {
			handler.failed(writer, "", 0, errors.NotAcceptable("fns: request version is not acceptable").WithMeta("version", handler.appVersion.String()))
			return
		}
	}
	// devIp
	devIp := r.Header.Get(httpDeviceIpHeader)
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

	// id
	id := r.Header.Get(httpRequestIdHeader)
	if id == "" {
		id = uid.UID()
	}

	ep, hasEndpoint := handler.endpoints[serviceName]
	if !hasEndpoint {
		handler.failed(writer, "", 0, errors.NotFound("fns: service was not found").WithMeta("service", serviceName))
		return
	}

	// timeout
	ctx := r.Context()
	var cancel context.CancelFunc
	timeout := r.Header.Get(httpRequestTimeoutHeader)
	if timeout != "" {
		timeoutMillisecond, parseTimeoutErr := strconv.ParseInt(timeout, 10, 64)
		if parseTimeoutErr != nil {
			handler.failed(writer, "", 0, errors.BadRequest("fns: X-Fns-Request-Timeout is not number").WithMeta("timeout", timeout))
			return
		}
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMillisecond)*time.Millisecond)
	}
	// request
	var req Request
	if internal {
		ir := &internalRequest{}
		decodeErr := json.Unmarshal(body, ir)
		if decodeErr != nil {
			handler.failed(writer, "", 0, errors.NotAcceptable("fns: decode internal request failed").WithCause(decodeErr))
			if cancel != nil {
				cancel()
			}
			return
		}
		req = NewRequest(
			ctx,
			devId,
			serviceName,
			fnName,
			NewArgument(ir.Body),
			WithHttpRequestHeader(r.Header),
			WithDeviceIp(devIp),
			WithRequestId(id),
			WithInternalRequest(),
			WithRequestTrunk(ir.Trunk),
			WithRequestUser(ir.User.Id(), ir.User.Attributes()),
		)
	} else {
		req = NewRequest(
			ctx,
			devId,
			serviceName,
			fnName,
			NewArgument(body),
			WithHttpRequestHeader(r.Header),
			WithDeviceIp(devIp),
			WithRequestId(id),
		)
	}
	// send
	handleBegAT := time.Time{}
	latency := time.Duration(0)
	if handler.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	result, hasResult, requestErr := ep.RequestSync(ctx, req)
	if cancel != nil {
		cancel()
	}
	if handler.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	// write
	if requestErr != nil {
		handler.failed(writer, id, latency, requestErr)
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
					handler.failed(writer, id, latency, errors.Warning("fns: encoding result failed").WithCause(encodeErr))
				} else {
					handler.succeed(writer, id, latency, p)
				}
				break
			}
		} else {
			handler.succeed(writer, id, latency, []byte{'{', '}'})
		}
	}
}

func (handler *servicesHandler) handleDocuments(writer http.ResponseWriter, r *http.Request) {
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

func (handler *servicesHandler) handleOpenapi(writer http.ResponseWriter, r *http.Request) {
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
		handler.failed(writer, id, latency, cause)
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

func (handler *servicesHandler) failed(writer http.ResponseWriter, id string, latency time.Duration, cause errors.CodeError) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	if id != "" {
		writer.Header().Set(httpRequestIdHeader, id)
	}
	if handler.log.DebugEnabled() {
		writer.Header().Set(httpHandleLatencyHeader, latency.String())
	}
	writer.WriteHeader(cause.Code())
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

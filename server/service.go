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

package server

import (
	stdjson "encoding/json"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	httpIdHeader      = "X-Fns-Request-Id"
	httpLatencyHeader = "X-Fns-Latency"
)

type serviceHandler struct {
	log       logs.Logger
	counter   sync.WaitGroup
	version   versions.Version
	endpoints map[string]service.Endpoint
}

func (h *serviceHandler) Name() (name string) {
	name = "service"
	return
}

func (h *serviceHandler) Build(options *HandlerOptions) (err error) {
	h.log = options.Log
	h.counter = sync.WaitGroup{}
	h.version = options.AppVersion
	h.endpoints = make(map[string]service.Endpoint)
	for _, endpoint := range options.DeployedEndpoints.Deployed() {
		h.endpoints[endpoint.Name()] = endpoint
	}
	return
}

func (h *serviceHandler) Accept(request *http.Request) (ok bool) {
	if request.Method != http.MethodPost {
		return
	}
	if request.Header.Get(httpContentType) != httpContentTypeJson {
		return
	}
	if len(strings.Split(request.URL.Path, "/")) != 3 {
		return
	}
	ok = true
	return
}

func (h *serviceHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	pathItems := strings.Split(request.URL.Path, "/")
	if len(pathItems) != 3 {
		h.failed(writer, "", 0, errors.BadRequest("fns: invalid request url path"))
		return
	}
	serviceName := pathItems[1]
	fnName := pathItems[2]
	body, readBodyErr := ioutil.ReadAll(request.Body)
	if readBodyErr != nil {
		h.failed(writer, "", 0, errors.BadRequest("fns: read body failed").WithCause(readBodyErr))
		return
	}
	id := request.Header.Get("X-Fns-Request-Id")
	if id == "" {
		id = uid.UID()
	}
	remoteIp := request.Header.Get("X-Real-Ip")
	if remoteIp == "" {
		forwarded := request.Header.Get("X-Forwarded-For")
		if forwarded != "" {
			forwardedIps := strings.Split(forwarded, ",")
			remoteIp = strings.TrimSpace(forwardedIps[len(forwardedIps)-1])
		}
	}
	if remoteIp == "" {
		remoteIp = request.RemoteAddr
		if remoteIp != "" {
			if strings.Index(remoteIp, ".") > 0 && strings.Index(remoteIp, ":") > 0 {
				remoteIp = remoteIp[0:strings.Index(remoteIp, ":")]
			}
		}
	}
	version := request.Header.Get("X-Fns-Version")
	if version != "" {
		leftVersion := versions.Version{}
		rightVersion := versions.Version{}
		var parseVersionErr error
		versionRange := strings.Split(version, ",")
		leftVersionValue := strings.TrimSpace(versionRange[0])
		if leftVersionValue != "" {
			leftVersion, parseVersionErr = versions.Parse(leftVersionValue)
			if parseVersionErr != nil {
				h.failed(writer, "", 0, errors.BadRequest("fns: read request version failed").WithCause(parseVersionErr))
				return
			}
		}
		if len(versionRange) > 1 {
			rightVersionValue := strings.TrimSpace(versionRange[1])
			if rightVersionValue != "" {
				rightVersion, parseVersionErr = versions.Parse(rightVersionValue)
				if parseVersionErr != nil {
					h.failed(writer, "", 0, errors.BadRequest("fns: read request version failed").WithCause(parseVersionErr))
					return
				}
			}
		}
		if !h.version.Between(leftVersion, rightVersion) {
			h.failed(writer, "", 0, errors.NotAcceptable("fns: request version is not acceptable").WithMeta("version", h.version.String()))
			return
		}
	}

	h.counter.Add(1)
	handleBegAT := time.Time{}
	latency := time.Duration(0)
	if h.log.DebugEnabled() {
		handleBegAT = time.Now()
	}

	endpoint, hasEndpoint := h.endpoints[serviceName]
	if !hasEndpoint {
		h.failed(writer, "", 0, errors.NotFound("service was not found").WithMeta("service", serviceName))
		return
	}

	endpointRequest := service.NewRequest(
		request.Context(),
		serviceName,
		fnName,
		service.NewArgument(body),
		service.WithHttpRequestHeader(request.Header),
		service.WithRemoteClientIp(remoteIp),
		service.WithRequestId(id),
	)

	result := endpoint.Request(request.Context(), endpointRequest)

	if h.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	resultValue, hasResultValue, requestErr := result.Value(request.Context())
	if requestErr != nil {
		h.failed(writer, id, latency, requestErr)
	} else {
		if hasResultValue {
			switch resultValue.(type) {
			case []byte:
				h.succeed(writer, id, latency, resultValue.([]byte))
				break
			case json.RawMessage:
				h.succeed(writer, id, latency, resultValue.(json.RawMessage))
				break
			case stdjson.RawMessage:
				h.succeed(writer, id, latency, resultValue.(stdjson.RawMessage))
				break
			default:
				p, encodeErr := json.Marshal(resultValue)
				if encodeErr != nil {
					h.failed(writer, id, latency, errors.Warning("fns: encoding result failed").WithCause(encodeErr))
				} else {
					h.succeed(writer, id, latency, p)
				}
				break
			}
		} else {
			h.succeed(writer, id, latency, []byte{'{', '}'})
		}
	}
	h.counter.Done()
	return
}

func (h *serviceHandler) Close() {
	h.counter.Wait()
}

func (h *serviceHandler) succeed(writer http.ResponseWriter, id string, latency time.Duration, body []byte) {
	writer.Header().Set(httpServerHeader, httpServerHeaderValue)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	if id != "" {
		writer.Header().Set(httpIdHeader, id)
	}
	if h.log.DebugEnabled() {
		writer.Header().Set(httpLatencyHeader, latency.String())
	}
	writer.WriteHeader(http.StatusOK)
	if body == nil || len(body) == 0 {
		return
	}
	_, _ = writer.Write(body)
}

func (h *serviceHandler) failed(writer http.ResponseWriter, id string, latency time.Duration, codeErr errors.CodeError) {
	writer.Header().Set(httpServerHeader, httpServerHeaderValue)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	if id != "" {
		writer.Header().Set(httpIdHeader, id)
	}
	if h.log.DebugEnabled() {
		writer.Header().Set(httpLatencyHeader, latency.String())
	}
	writer.WriteHeader(codeErr.Code())
	p, _ := json.Marshal(codeErr)
	_, _ = writer.Write(p)
}

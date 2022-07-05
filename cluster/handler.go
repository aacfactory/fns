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

package cluster

import (
	stdjson "encoding/json"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"io/ioutil"
	"net/http"
	"sync"
)

const (
	httpContentType        = "Content-Type"
	httpContentTypeProxy   = "application/fns+proxy"
	httpContentTypeCluster = "application/fns+cluster"
	httpContentTypeJson    = "application/json"

	joinPath  = "/cluster/join"
	leavePath = "/cluster/leave"
)

type HandlerOptions struct {
	Log       logs.Logger
	Endpoints service.Endpoints
	Cluster   *Manager
}

func NewHandler(options HandlerOptions) *Handler {
	return &Handler{
		log: options.Log.With("fns", "cluster"),
		proxy: &proxyHandler{
			log:       options.Log.With("fns", "cluster").With("cluster", "proxy"),
			counter:   sync.WaitGroup{},
			endpoints: options.Endpoints,
		},
		member: &clusterHandler{
			log:     options.Log.With("fns", "cluster").With("cluster", "members"),
			manager: options.Cluster,
		},
	}
}

type Handler struct {
	log    logs.Logger
	proxy  *proxyHandler
	member *clusterHandler
}

func (handler *Handler) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
	if request.Method != http.MethodPost {
		return
	}
	contentType := request.Header.Get(httpContentType)
	switch contentType {
	case httpContentTypeProxy:
		ok = true
		handler.proxy.Handle(writer, request)
	case httpContentTypeCluster:
		ok = true
		handler.member.Handle(writer, request)
	default:
		return
	}
	return
}

func (handler *Handler) Close() {
	handler.proxy.Close()
	return
}

type proxyHandler struct {
	log       logs.Logger
	counter   sync.WaitGroup
	endpoints service.Endpoints
}

func (handler *proxyHandler) Handle(writer http.ResponseWriter, request *http.Request) {
	r, requestErr := newRequest(request)
	if requestErr != nil {
		handler.failed(writer, nil, requestErr)
		return
	}
	handler.counter.Add(1)
	ctx := request.Context()
	ctx = service.SetRequest(ctx, r)
	ctx = service.SetTracer(ctx)
	result, handleErr := handler.endpoints.Handle(ctx, r)
	var span service.Span
	tracer, hasTracer := service.GetTracer(ctx)
	if hasTracer {
		span = tracer.Span()
	}
	if handleErr == nil {
		switch result.(type) {
		case []byte:
			handler.succeed(writer, span, result.([]byte))
			break
		case json.RawMessage:
			handler.succeed(writer, span, result.(json.RawMessage))
			break
		case stdjson.RawMessage:
			handler.succeed(writer, span, result.(stdjson.RawMessage))
			break
		default:
			p, encodeErr := json.Marshal(result)
			if encodeErr != nil {
				handler.failed(writer, span, errors.Warning("fns: encoding result failed").WithCause(encodeErr))
			} else {
				handler.succeed(writer, span, p)
			}
			break
		}
	} else {
		handler.failed(writer, span, handleErr)
	}
	handler.counter.Done()
	return
}

func (handler *proxyHandler) succeed(writer http.ResponseWriter, span service.Span, body []byte) {
	var spanData []byte = nil
	if span != nil {
		p, err := json.Marshal(span)
		if err == nil {
			spanData = p
		}
	}
	resp := &response{
		SpanData: spanData,
		Data:     body,
	}
	p, encodeErr := json.Marshal(resp)
	if encodeErr != nil {
		if handler.log.WarnEnabled() {
			handler.log.Warn().Cause(encodeErr).With("cluster", "handler").Message("fns: encode internal response failed")
		}
		handler.failed(writer, span, errors.Map(encodeErr))
		return
	}
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(p)
}

func (handler *proxyHandler) failed(writer http.ResponseWriter, span service.Span, codeErr errors.CodeError) {
	var spanData []byte = nil
	if span != nil {
		p, err := json.Marshal(span)
		if err != nil {
			if handler.log.WarnEnabled() {
				handler.log.Warn().Cause(err).With("cluster", "handler").Message("fns: encode span failed")
			}
			handler.failed(writer, span, errors.Map(err))
			return
		}
		spanData = p
	}
	p, encodeErr := json.Marshal(codeErr)
	if encodeErr != nil {
		if handler.log.WarnEnabled() {
			handler.log.Warn().Cause(encodeErr).With("cluster", "handler").Message("fns: encode error failed")
		}
		handler.failed(writer, span, errors.Map(encodeErr))
		return
	}
	resp := &response{
		SpanData: spanData,
		Data:     p,
	}
	p, encodeErr = json.Marshal(resp)
	if encodeErr != nil {
		if handler.log.WarnEnabled() {
			handler.log.Warn().Cause(encodeErr).With("cluster", "handler").Message("fns: encode internal response failed")
		}
		handler.failed(writer, span, errors.Map(encodeErr))
		return
	}
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(codeErr.Code())
	_, _ = writer.Write(p)
}

func (handler *proxyHandler) Close() {
	handler.counter.Wait()
}

type clusterHandler struct {
	log     logs.Logger
	manager *Manager
}

func (handler *clusterHandler) Handle(writer http.ResponseWriter, request *http.Request) {
	devMode := request.Header.Get("X-Fns-DevMode") == "true"
	bodyRaw, bodyErr := ioutil.ReadAll(request.Body)
	if bodyErr != nil {
		handler.failed(writer, errors.BadRequest("fns: read body failed").WithCause(bodyErr))
		return
	}
	body, bodyValid := decodeRequestBody(bodyRaw)
	if !bodyValid {
		handler.failed(writer, errors.NotAcceptable("fns: invalid body"))
		return
	}
	switch request.URL.Path {
	case joinPath:
		result, handleErr := handler.handleJoin(body, devMode)
		if handleErr == nil {
			handler.succeed(writer, result)
		} else {
			handler.failed(writer, handleErr)
		}
	case leavePath:
		handleErr := handler.handleLeave(body, devMode)
		if handleErr == nil {
			handler.succeed(writer, nil)
		} else {
			handler.failed(writer, handleErr)
		}
	default:
		handler.failed(writer, errors.NotFound("fns: not found").WithMeta("uri", request.URL.Path))
		return
	}
	return
}

func (handler *clusterHandler) handleJoin(body []byte, devMode bool) (result []byte, err errors.CodeError) {
	n := &node{}
	decodeErr := json.Unmarshal(body, n)
	if decodeErr != nil {
		err = errors.Warning("fns: decode body failed").WithCause(decodeErr)
		return
	}
	if n.Id_ == handler.manager.node.Id_ || n.Address == handler.manager.node.Address {
		result = []byte{'[', ']'}
		return
	}
	nodes := make([]*node, 0, 1)
	nodes = append(nodes, handler.manager.node)
	nodes = append(nodes, handler.manager.registrations.members()...)
	p, encodeErr := json.Marshal(nodes)
	if encodeErr != nil {
		err = errors.Warning("fns: encode result failed").WithCause(encodeErr)
		return
	}
	result = p
	if !devMode {
		handler.manager.registrations.register(n)
	}

	return
}

func (handler *clusterHandler) handleLeave(body []byte, devMode bool) (err errors.CodeError) {
	if devMode {
		return
	}
	n := &node{}
	decodeErr := json.Unmarshal(body, n)
	if decodeErr != nil {
		err = errors.Warning("fns: decode body failed").WithCause(decodeErr)
		return
	}
	handler.manager.registrations.deregister(n)
	return
}

func (handler *clusterHandler) succeed(writer http.ResponseWriter, body []byte) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(http.StatusOK)
	if body != nil && len(body) > 0 {
		_, _ = writer.Write(body)
	}
}

func (handler *clusterHandler) failed(writer http.ResponseWriter, codeErr errors.CodeError) {
	p, encodeErr := json.Marshal(codeErr)
	if encodeErr != nil {
		if handler.log.WarnEnabled() {
			handler.log.Warn().Cause(encodeErr).With("cluster", "handler").Message("fns: encode internal response failed")
		}
		handler.failed(writer, errors.Map(encodeErr))
		return
	}
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(codeErr.Code())
	_, _ = writer.Write(p)
}

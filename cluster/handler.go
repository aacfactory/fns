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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
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
	Log           logs.Logger
	Endpoints     service.Endpoints
	Registrations *RegistrationsManager
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
			log:           options.Log.With("fns", "cluster").With("cluster", "members"),
			registrations: options.Registrations,
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
	r, requestErr := service.NewRequest(request)
	if requestErr != nil {
		handler.failed(writer, requestErr)
		return
	}
	handler.counter.Add(1)
	ctx := request.Context()
	ctx = service.SetRequest(ctx, r)
	ctx = service.SetTracer(ctx)
	result, handleErr := handler.endpoints.Handle(request.Context(), r)
	if handleErr == nil {
		tracer, _ := service.GetTracer(ctx)
		handler.succeed(writer, tracer.Span(), result)
	} else {
		handler.failed(writer, handleErr)
	}
	handler.counter.Done()
	return
}

func (handler *proxyHandler) succeed(writer http.ResponseWriter, span service.Span, body []byte) {
	resp := &response{
		Span: span,
		Data: body,
	}
	p, encodeErr := json.Marshal(resp)
	if encodeErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: encode response to json failed").WithCause(encodeErr)))
	}
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(p)
}

func (handler *proxyHandler) failed(writer http.ResponseWriter, codeErr errors.CodeError) {
	p, encodeErr := json.Marshal(codeErr)
	if encodeErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: encode code error to json failed").WithCause(encodeErr).WithCause(codeErr)))
	}
	resp := &response{
		Span: nil,
		Data: p,
	}
	p, encodeErr = json.Marshal(resp)
	if encodeErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: encode response to json failed").WithCause(encodeErr).WithCause(codeErr)))
	}
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(codeErr.Code())
	_, _ = writer.Write(p)
}

func (handler *proxyHandler) Close() {
	handler.counter.Wait()
}

type clusterHandler struct {
	log           logs.Logger
	registrations *RegistrationsManager
}

func (handler *clusterHandler) Handle(writer http.ResponseWriter, request *http.Request) {
	r, requestErr := service.NewRequest(request)
	if requestErr != nil {
		handler.failed(writer, requestErr)
		return
	}
	sn, fn := r.Fn()
	if sn != "cluster" {
		handler.failed(writer, errors.NotAcceptable("fns: invalid url path"))
		return
	}
	switch fn {
	case "join":
		result, handleErr := handler.handleJoin(r)
		if handler == nil {
			handler.succeed(writer, result)
		} else {
			handler.failed(writer, handleErr)
		}
	case "leave":
		result, handleErr := handler.handleLeave(r)
		if handler == nil {
			handler.succeed(writer, result)
		} else {
			handler.failed(writer, handleErr)
		}
	default:
		handler.failed(writer, errors.NotAcceptable("fns: invalid url path"))
		return
	}
	return
}

type joinResult struct {
	Node    *Node   `json:"node"`
	Members []*Node `json:"members"`
}

func (handler *clusterHandler) handleJoin(r service.Request) (result []byte, err errors.CodeError) {
	// todo
	return
}

func (handler *clusterHandler) handleLeave(r service.Request) (result []byte, err errors.CodeError) {
	// todo
	return
}

func (handler *clusterHandler) succeed(writer http.ResponseWriter, body []byte) {
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(body)
}

func (handler *clusterHandler) failed(writer http.ResponseWriter, codeErr errors.CodeError) {
	p, encodeErr := json.Marshal(codeErr)
	if encodeErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: encode code error to json failed").WithCause(encodeErr).WithCause(codeErr)))
	}
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(codeErr.Code())
	_, _ = writer.Write(p)
}

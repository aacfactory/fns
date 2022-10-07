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
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	httpIdHeader      = "X-Fns-Request-Id"
	httpLatencyHeader = "X-Fns-Latency"
)

func NewServiceHandler() (h Handler) {
	h = &serviceHandler{
		log:       nil,
		counter:   sync.WaitGroup{},
		endpoints: nil,
	}
	return
}

type serviceHandler struct {
	log       logs.Logger
	counter   sync.WaitGroup
	endpoints service.Endpoints
}

func (h *serviceHandler) Name() (name string) {
	name = "service"
	return
}

func (h *serviceHandler) Build(options *HandlerOptions) (err error) {
	h.log = options.Log.With("fns", "handler").With("handle", "service")
	h.endpoints = options.Endpoints
	return
}

func (h *serviceHandler) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
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
	r, requestErr := service.NewRequest(request)
	if requestErr != nil {
		h.failed(writer, "", 0, requestErr)
		return
	}
	h.counter.Add(1)
	handleBegAT := time.Time{}
	latency := time.Duration(0)
	if h.log.DebugEnabled() {
		handleBegAT = time.Now()
	}
	result, handleErr := h.endpoints.Handle(request.Context(), r)
	if h.log.DebugEnabled() {
		latency = time.Now().Sub(handleBegAT)
	}
	if handleErr == nil {
		if result == nil {
			h.succeed(writer, r.Id(), latency, nil)
		} else {
			switch result.(type) {
			case []byte:
				h.succeed(writer, r.Id(), latency, result.([]byte))
				break
			case json.RawMessage:
				h.succeed(writer, r.Id(), latency, result.(json.RawMessage))
				break
			case stdjson.RawMessage:
				h.succeed(writer, r.Id(), latency, result.(stdjson.RawMessage))
				break
			default:
				p, encodeErr := json.Marshal(result)
				if encodeErr != nil {
					h.failed(writer, r.Id(), latency, errors.Warning("fns: encoding result failed").WithCause(encodeErr))
				} else {
					h.succeed(writer, r.Id(), latency, p)
				}
				break
			}
		}
	} else {
		h.failed(writer, r.Id(), latency, handleErr)
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

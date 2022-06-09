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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"net/http"
	"sync"
)

const (
	// path
	httpHealthPath      = "/health"
	httpDocumentRawPath = "/documents/raw"
	httpDocumentOASPath = "/documents/oas.json"

	// header
	httpServerHeader          = "Server"
	httpServerHeaderValue     = "FNS"
	httpContentType           = "Content-Type"
	httpContentTypeProxy      = "application/fns+proxy"
	httpContentTypeJson       = "application/json"
	httpAuthorizationHeader   = "Authorization"
	httpConnectionHeader      = "Connection"
	httpConnectionHeaderClose = "close"
	httpIdHeader              = "X-Fns-Request-Id"
	httpLatencyHeader         = "X-Fns-Latency"
	httpXForwardedFor         = "X-Forwarded-For"
	httpXRealIp               = "X-Real-Ip"
)

type Handler struct {
	log     logs.Logger
	counter sync.WaitGroup
	sh      service.Endpoints
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

}

func (h *Handler) succeed(response http.ResponseWriter, body []byte) {
	response.Header().Set(httpServerHeader, httpServerHeaderValue)
	response.Header().Set(httpContentType, httpContentTypeJson)
	response.WriteHeader(200)
	if body == nil || len(body) == 0 {
		return
	}
	_, _ = response.Write(body)
}

func (h *Handler) failed(response http.ResponseWriter, codeErr errors.CodeError) {
	response.Header().Set(httpServerHeader, httpServerHeaderValue)
	response.Header().Set(httpContentType, httpContentTypeJson)
	response.WriteHeader(codeErr.Code())
	p, _ := json.Marshal(codeErr)
	_, _ = response.Write(p)
}

func (h *Handler) serviceDocuments(response http.ResponseWriter) {
	raw, rawErr := h.document.Json()
	if rawErr != nil {
		h.failed(response, rawErr)
		return
	}
	h.succeed(response, raw)
}

func (h *Handler) openAPIDocument(response http.ResponseWriter) {
	raw, rawErr := h.document.OAS()
	if rawErr != nil {
		h.failed(response, rawErr)
		return
	}
	h.succeed(response, raw)
}

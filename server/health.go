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
	"fmt"
	"net/http"
	"time"
)

const (
	httpHealthPath = "/health"
)

type healthHandler struct {
	appId    string
	appName  string
	version  string
	launchAT string
}

func (h *healthHandler) Name() (name string) {
	name = "health"
	return
}

func (h *healthHandler) Build(options *HandlerOptions) (err error) {
	h.appId = options.AppId
	h.appName = options.AppName
	h.version = options.AppVersion
	h.launchAT = time.Now().Format(time.RFC3339)
	return
}

func (h *healthHandler) Accept(request *http.Request) (ok bool) {
	ok = request.Method == http.MethodGet && request.URL.Path == httpHealthPath
	return
}

func (h *healthHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	body := fmt.Sprintf(
		"{\"name\":\"%s\", \"id\":\"%s\", \"version\":\"%s\", \"launch\":\"%s\", \"now\":\"%s\"}",
		h.appName, h.appId, h.version, h.launchAT, time.Now().Format(time.RFC3339),
	)
	writer.Header().Set(httpServerHeader, httpServerHeaderValue)
	writer.Header().Set(httpContentType, httpContentTypeJson)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(body))
	return
}

func (h *healthHandler) Close() {
}

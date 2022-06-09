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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/json"
	"net/http"
	"time"
)

const (
	httpHealthPath            = "/health"
	httpConnectionHeader      = "Connection"
	httpConnectionHeaderClose = "close"
)

type HealthHandlerOptions struct {
	AppId   string
	Version string
	Running *commons.SafeFlag
}

func NewHealthHandler(options HealthHandlerOptions) (h Handler) {
	h = &healthHandler{
		appId:       options.AppId,
		version:     options.Version,
		running:     options.Running,
		unavailable: json.UnsafeMarshal(errors.Unavailable("fns: service is unavailable").WithMeta("fns", "http")),
	}
	return
}

type healthHandler struct {
	appId       string
	version     string
	running     *commons.SafeFlag
	unavailable []byte
}

func (h *healthHandler) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
	if h.running.IsOff() {
		writer.Header().Set(httpConnectionHeader, httpConnectionHeaderClose)
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write(h.unavailable)
		ok = true
		return
	}
	if request.Method != http.MethodGet {
		return
	}
	switch request.URL.Path {
	case "", "/", httpHealthPath:
		ok = true
		body := fmt.Sprintf(
			"{\"appId\":\"%s\", \"version\":\"%s\", \"running\":\"%v\", \"now\":\"%s\"}",
			h.appId, h.version, h.running.IsOn(), time.Now().Format(time.RFC3339),
		)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(body))
		break
	default:
		return
	}
	return
}

func (h *healthHandler) Close() {
}

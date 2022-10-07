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
	"github.com/aacfactory/fns/internal/configure"
	"github.com/aacfactory/fns/internal/cors"
	"net/http"
	"sort"
)

func NewCorsHandler() (h Handler) {
	h = &corsHandler{}
	return
}

type corsHandler struct {
	cors *cors.Cors
}

func (h *corsHandler) Name() (name string) {
	name = "cors"
	return
}

func (h *corsHandler) Build(options *HandlerOptions) (err error) {
	config := &configure.Cors{}
	has, getErr := options.Config.Get("server.cors", config)
	if getErr != nil {
		err = fmt.Errorf("build cors handler failed, %v", getErr)
		return
	}
	if has {
		allowedOrigins := config.AllowedOrigins
		if allowedOrigins == nil {
			allowedOrigins = make([]string, 0, 1)
		}
		if len(allowedOrigins) == 0 {
			allowedOrigins = append(allowedOrigins, "*")
		}
		allowedHeaders := config.AllowedHeaders
		if allowedHeaders == nil {
			allowedHeaders = make([]string, 0, 1)
		}
		if sort.SearchStrings(allowedHeaders, "Connection") < 0 {
			allowedHeaders = append(allowedHeaders, "Connection")
		}
		if sort.SearchStrings(allowedHeaders, "Upgrade") < 0 {
			allowedHeaders = append(allowedHeaders, "Upgrade")
		}
		if sort.SearchStrings(allowedHeaders, "X-Forwarded-For") < 0 {
			allowedHeaders = append(allowedHeaders, "X-Forwarded-For")
		}
		if sort.SearchStrings(allowedHeaders, "X-Real-Ip") < 0 {
			allowedHeaders = append(allowedHeaders, "X-Real-Ip")
		}
		exposedHeaders := config.ExposedHeaders
		if exposedHeaders == nil {
			exposedHeaders = make([]string, 0, 1)
		}
		exposedHeaders = append(exposedHeaders, "X-Fns-Request-Id", "X-Fns-Latency", "Connection", "Server")
		h.cors = cors.New(cors.Options{
			AllowedOrigins:       allowedOrigins,
			AllowedMethods:       []string{http.MethodGet, http.MethodPost},
			AllowedHeaders:       allowedHeaders,
			ExposedHeaders:       exposedHeaders,
			MaxAge:               config.MaxAge,
			AllowCredentials:     config.AllowCredentials,
			AllowPrivateNetwork:  true,
			OptionsPassthrough:   false,
			OptionsSuccessStatus: http.StatusNoContent,
		})
	} else {
		h.cors = cors.AllowAll()
	}
	return
}

func (h *corsHandler) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
	ok = h.cors.Handle(writer, request)
	return
}

func (h *corsHandler) Close() {
}

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
	"github.com/aacfactory/fns/internal/cors"
	"net/http"
	"sort"
)

type CorsHandlerOptions struct {
	Customized       bool
	AllowedOrigins   []string `copy:"AllowedOrigins"`
	AllowedHeaders   []string `copy:"AllowedHeaders"`
	ExposedHeaders   []string `copy:"ExposedHeaders"`
	AllowCredentials bool     `copy:"AllowCredentials"`
	MaxAge           int      `copy:"MaxAge"`
}

func NewCorsHandler(options CorsHandlerOptions) (h Handler) {
	var c *cors.Cors
	if options.Customized == false {
		c = cors.AllowAll()
	} else {
		allowedOrigins := options.AllowedOrigins
		if allowedOrigins == nil {
			allowedOrigins = make([]string, 0, 1)
		}
		if len(allowedOrigins) == 0 {
			allowedOrigins = append(allowedOrigins, "*")
		}
		allowedHeaders := options.AllowedHeaders
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
		exposedHeaders := options.ExposedHeaders
		if exposedHeaders == nil {
			exposedHeaders = make([]string, 0, 1)
		}
		exposedHeaders = append(exposedHeaders, httpIdHeader, httpLatencyHeader, httpConnectionHeader, "Server")
		opt := cors.Options{
			AllowedOrigins:       options.AllowedOrigins,
			AllowedMethods:       []string{http.MethodGet, http.MethodPost},
			AllowedHeaders:       allowedHeaders,
			ExposedHeaders:       exposedHeaders,
			MaxAge:               options.MaxAge,
			AllowCredentials:     options.AllowCredentials,
			AllowPrivateNetwork:  true,
			OptionsPassthrough:   false,
			OptionsSuccessStatus: http.StatusNoContent,
		}
		c = cors.New(opt)
	}
	h = &corsHandler{
		cors: c,
	}
	return
}

type corsHandler struct {
	cors *cors.Cors
}

func (h *corsHandler) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
	ok = h.cors.Handle(writer, request)
	return
}

func (h *corsHandler) Close() {
}

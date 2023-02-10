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
	"github.com/aacfactory/fns/internal/configure"
	"github.com/rs/cors"
	"net/http"
	"sort"
)

func NewCorsHandler(config *configure.Cors) (h *cors.Cors) {
	if config == nil {
		config = &configure.Cors{
			AllowedOrigins:   []string{"*"},
			AllowedHeaders:   []string{"*"},
			ExposedHeaders:   nil,
			AllowCredentials: false,
			MaxAge:           0,
		}
	}
	if config.AllowedOrigins == nil || len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}
	if config.AllowedHeaders == nil || len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = make([]string, 0, 1)
		config.AllowedHeaders = append(config.AllowedHeaders, "*")
	}
	if config.AllowedHeaders[0] != "*" {
		if sort.SearchStrings(config.AllowedHeaders, "Connection") < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, "Connection")
		}
		if sort.SearchStrings(config.AllowedHeaders, "Upgrade") < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, "Upgrade")
		}
		if sort.SearchStrings(config.AllowedHeaders, "X-Forwarded-For") < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, "X-Forwarded-For")
		}
		if sort.SearchStrings(config.AllowedHeaders, "X-Real-Ip") < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, "X-Real-Ip")
		}
		if sort.SearchStrings(config.AllowedHeaders, "X-Fns-Client-Id") < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, "X-Fns-Client-Id")
		}
		if sort.SearchStrings(config.AllowedHeaders, "X-Fns-Request-Timeout") < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, "X-Fns-Request-Timeout")
		}
	}
	if config.ExposedHeaders == nil {
		config.ExposedHeaders = make([]string, 0, 1)
	}
	config.ExposedHeaders = append(config.ExposedHeaders, "X-Fns-Request-Id", "X-Fns-Latency", "Connection", "Server")
	h = cors.New(cors.Options{
		AllowedOrigins:         config.AllowedOrigins,
		AllowOriginFunc:        nil,
		AllowOriginRequestFunc: nil,
		AllowedMethods:         []string{http.MethodGet, http.MethodPost},
		AllowedHeaders:         config.AllowedHeaders,
		ExposedHeaders:         config.ExposedHeaders,
		MaxAge:                 config.MaxAge,
		AllowCredentials:       config.AllowCredentials,
		AllowPrivateNetwork:    config.AllowPrivateNetwork,
		OptionsPassthrough:     false,
		OptionsSuccessStatus:   204,
		Debug:                  false,
	})
	return
}

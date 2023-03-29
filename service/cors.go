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

package service

import (
	"github.com/rs/cors"
	"net/http"
	"sort"
)

func newCorsHandler(config *CorsConfig) (h *cors.Cors) {
	if config == nil {
		config = &CorsConfig{
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
		if sort.SearchStrings(config.AllowedHeaders, httpConnectionHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpConnectionHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpUpgradeHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpUpgradeHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpXForwardedForHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpXForwardedForHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpDeviceIpHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpDeviceIpHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpDeviceIdHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpDeviceIdHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpRequestIdHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpRequestIdHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpRequestSignatureHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpRequestSignatureHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpRequestTimeoutHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpRequestTimeoutHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpRequestVersionsHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpRequestVersionsHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpDevModeHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpDevModeHeader)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpCacheControlIfNonMatch) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpCacheControlIfNonMatch)
		}
		if sort.SearchStrings(config.AllowedHeaders, httpPragmaHeader) < 0 {
			config.AllowedHeaders = append(config.AllowedHeaders, httpPragmaHeader)
		}
	}
	if config.ExposedHeaders == nil {
		config.ExposedHeaders = make([]string, 0, 1)
	}
	config.ExposedHeaders = append(
		config.ExposedHeaders,
		httpAppIdHeader, httpAppNameHeader, httpAppVersionHeader,
		httpRequestIdHeader, httpRequestSignatureHeader, httpHandleLatencyHeader,
		httpCacheControlHeader, httpETagHeader, httpClearSiteData, httpResponseRetryAfter,
	)
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

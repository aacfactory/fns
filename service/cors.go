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
	"github.com/aacfactory/fns/service/transports"
)

const (
	httpContentType            = "Content-Type"
	httpContentTypeJson        = "application/json"
	httpConnectionHeader       = "Connection"
	httpUpgradeHeader          = "Upgrade"
	httpCloseHeader            = "close"
	httpCacheControlHeader     = "Cache-Control"
	httpPragmaHeader           = "Pragma"
	httpETagHeader             = "ETag"
	httpCacheControlIfNonMatch = "If-None-Match"
	httpClearSiteData          = "Clear-Site-Data"
	httpTrueClientIp           = "True-Client-Ip"
	httpXRealIp                = "X-Real-IP"
	httpXForwardedForHeader    = "X-Forwarded-For"
	httpRequestIdHeader        = "X-Fns-Request-Id"
	httpRequestSignatureHeader = "X-Fns-Request-Signature"
	httpRequestInternalHeader  = "X-Fns-Request-Internal"
	httpRequestTimeoutHeader   = "X-Fns-Request-Timeout"
	httpRequestVersionsHeader  = "X-Fns-Request-Version"
	httpHandleLatencyHeader    = "X-Fns-Handle-Latency"
	httpDeviceIdHeader         = "X-Fns-Device-Id"
	httpDeviceIpHeader         = "X-Fns-Device-Ip"
	httpDevModeHeader          = "X-Fns-Dev-Mode"
	httpResponseRetryAfter     = "Retry-After"
)

func newCorsHandler(config *transports.CorsConfig, handler transports.Handler) (cors transports.Handler) {
	if config == nil {
		config = &transports.CorsConfig{
			AllowedOrigins:      []string{"*"},
			AllowedHeaders:      []string{"*"},
			ExposedHeaders:      make([]string, 0, 1),
			AllowCredentials:    false,
			MaxAge:              86400,
			AllowPrivateNetwork: false,
		}
	}
	config.TryFillAllowedHeaders([]string{
		httpConnectionHeader, httpUpgradeHeader,
		httpXForwardedForHeader, httpTrueClientIp, httpXRealIp,
		httpDeviceIpHeader, httpDeviceIdHeader,
		httpRequestIdHeader,
		httpRequestSignatureHeader, httpRequestTimeoutHeader, httpRequestVersionsHeader,
		httpETagHeader, httpCacheControlIfNonMatch, httpPragmaHeader, httpClearSiteData, httpResponseRetryAfter,
	})
	config.TryFillExposedHeaders([]string{
		httpRequestIdHeader, httpRequestSignatureHeader, httpHandleLatencyHeader,
		httpCacheControlHeader, httpETagHeader, httpClearSiteData, httpResponseRetryAfter,
	})
	cors = config.Handler(handler)
	return
}

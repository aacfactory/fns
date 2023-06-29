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

package transports

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type CorsConfig struct {
	AllowedOrigins      []string `json:"allowedOrigins"`
	AllowedHeaders      []string `json:"allowedHeaders"`
	ExposedHeaders      []string `json:"exposedHeaders"`
	AllowCredentials    bool     `json:"allowCredentials"`
	MaxAge              int      `json:"maxAge"`
	AllowPrivateNetwork bool     `json:"allowPrivateNetwork"`
}

func CORS() Middleware {
	return &corsMiddleware{}
}

type corsMiddleware struct {
	allowedOrigins      []string
	allowedWOrigins     []wildcard
	allowedOriginsAll   bool
	allowedHeaders      []string
	allowedHeadersAll   bool
	allowedMethods      []string
	exposedHeaders      []string
	maxAge              int
	allowCredentials    bool
	allowPrivateNetwork bool
	handler             Handler
}

func (c *corsMiddleware) Name() string {
	return "cors"
}

func (c *corsMiddleware) Build(options MiddlewareOptions) (err error) {
	config := CorsConfig{}
	err = options.Config.As(&config)
	if err != nil {
		err = errors.Warning("fns: build cors middleware failed").WithCause(err)
		return
	}
	allowedOrigins := make([]string, 0, 1)
	allowedWOrigins := make([]wildcard, 0, 1)
	allowedOriginsAll := false
	if config.AllowedHeaders == nil {
		config.AllowedHeaders = make([]string, 0, 1)
	}
	if config.AllowedHeaders[0] != "*" {
		defaultAllowedHeaders := []string{
			ConnectionHeaderName, UpgradeHeaderName,
			XForwardedForHeaderName, TrueClientIpHeaderName, XRealIpHeaderName,
			DeviceIpHeaderName, DeviceIdHeaderName,
			RequestIdHeaderName,
			RequestInternalSignatureHeaderName, RequestTimeoutHeaderName, RequestVersionsHeaderName,
			ETagHeaderName, CacheControlHeaderIfNonMatch, ClearSiteDataHeaderName, ResponseRetryAfterHeaderName, SignatureHeaderName,
		}
		for _, header := range defaultAllowedHeaders {
			if sort.SearchStrings(config.AllowedHeaders, header) < 0 {
				config.AllowedHeaders = append(config.AllowedHeaders, header)
			}
		}
	}
	for _, origin := range config.AllowedOrigins {
		origin = strings.ToLower(origin)
		if origin == "*" {
			allowedOriginsAll = true
			allowedOrigins = nil
			allowedWOrigins = nil
			break
		} else if i := strings.IndexByte(origin, '*'); i >= 0 {
			w := wildcard{origin[0:i], origin[i+1:]}
			allowedWOrigins = append(allowedWOrigins, w)
		} else {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}
	var allowedHeaders []string
	allowedHeadersAll := false
	if len(config.AllowedHeaders) == 0 {
		allowedHeaders = []string{OriginHeaderName, AcceptHeaderName, ContentTypeHeaderName, XRequestedWithHeaderName}
	} else {
		allowedHeaders = make([]string, 0, 1)
		allowedHeaders = convert(append(config.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
		for _, h := range config.AllowedHeaders {
			if h == "*" {
				allowedHeadersAll = true
				allowedHeaders = nil
				break
			}
		}
	}
	if config.ExposedHeaders == nil {
		config.ExposedHeaders = make([]string, 0, 1)
	}
	defaultExposedHeaders := []string{
		RequestIdHeaderName, RequestInternalSignatureHeaderName, HandleLatencyHeaderName,
		CacheControlHeaderName, ETagHeaderName, ClearSiteDataHeaderName, ResponseRetryAfterHeaderName, ResponseCacheTTLHeaderName, SignatureHeaderName,
	}
	for _, header := range defaultExposedHeaders {
		if sort.SearchStrings(config.ExposedHeaders, header) < 0 {
			config.ExposedHeaders = append(config.ExposedHeaders, header)
		}
	}
	c.allowedOrigins = allowedOrigins
	c.allowedWOrigins = allowedWOrigins
	c.allowedOriginsAll = allowedOriginsAll
	c.allowedHeaders = allowedHeaders
	c.allowedHeadersAll = allowedHeadersAll
	c.allowedMethods = []string{http.MethodGet, http.MethodPost, http.MethodHead}
	c.exposedHeaders = convert(config.ExposedHeaders, http.CanonicalHeaderKey)
	c.maxAge = config.MaxAge
	c.allowCredentials = config.AllowCredentials
	c.allowPrivateNetwork = config.AllowPrivateNetwork
	return
}

func (c *corsMiddleware) Handler(next Handler) Handler {
	c.handler = next
	return c
}

func (c *corsMiddleware) Handle(w ResponseWriter, r *Request) {
	if bytex.ToString(r.Method()) == http.MethodOptions && r.Header().Get(AccessControlRequestMethodHeaderName) != "" {
		c.handlePreflight(w, r)
		w.SetStatus(http.StatusNoContent)
	} else {
		c.handleActualRequest(w, r)
		c.handler.Handle(w, r)
	}
}

func (c *corsMiddleware) handlePreflight(w ResponseWriter, r *Request) {
	headers := w.Header()
	origin := r.Header().Get(OriginHeaderName)

	if bytex.ToString(r.Method()) != http.MethodOptions {
		return
	}
	headers.Add(VaryHeaderName, OriginHeaderName)
	headers.Add(VaryHeaderName, AccessControlRequestMethodHeaderName)
	headers.Add(VaryHeaderName, AccessControlRequestHeadersHeaderName)
	if c.allowPrivateNetwork {
		headers.Add(VaryHeaderName, AccessControlRequestPrivateNetworkHeaderName)
	}

	if origin == "" {
		return
	}
	if !c.isOriginAllowed(origin) {
		return
	}

	reqMethod := r.Header().Get(AccessControlRequestMethodHeaderName)
	if !c.isMethodAllowed(reqMethod) {
		return
	}
	reqHeaders := parseHeaderList(r.Header().Get(AccessControlRequestHeadersHeaderName))
	if !c.areHeadersAllowed(reqHeaders) {
		return
	}
	if c.allowedOriginsAll {
		headers.Set(AccessControlAllowOriginHeaderName, "*")
	} else {
		headers.Set(AccessControlAllowOriginHeaderName, origin)
	}
	headers.Set(AccessControlAllowMethodsHeaderName, strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {
		headers.Set(AccessControlAllowHeadersHeaderName, strings.Join(reqHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set(AccessControlAllowCredentialsHeaderName, "true")
	}
	if c.allowPrivateNetwork && r.Header().Get(AccessControlRequestPrivateNetworkHeaderName) == "true" {
		headers.Set(AccessControlAllowPrivateNetworkHeaderName, "true")
	}
	if c.maxAge > 0 {
		headers.Set(AccessControlMaxAgeHeaderName, strconv.Itoa(c.maxAge))
	}
}

func (c *corsMiddleware) handleActualRequest(w ResponseWriter, r *Request) {
	headers := w.Header()
	origin := r.Header().Get(OriginHeaderName)

	headers.Add(VaryHeaderName, OriginHeaderName)
	if origin == "" {
		return
	}
	if !c.isOriginAllowed(origin) {
		return
	}

	if !c.isMethodAllowed(bytex.ToString(r.Method())) {
		return
	}
	if c.allowedOriginsAll {
		headers.Set(AccessControlAllowOriginHeaderName, "*")
	} else {
		headers.Set(AccessControlAllowOriginHeaderName, origin)
	}
	if len(c.exposedHeaders) > 0 {
		headers.Set(AccessControlExposeHeadersHeaderName, strings.Join(c.exposedHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set(AccessControlAllowCredentialsHeaderName, "true")
	}
}

func (c *corsMiddleware) isOriginAllowed(origin string) bool {
	if c.allowedOriginsAll {
		return true
	}
	origin = strings.ToLower(origin)
	for _, o := range c.allowedOrigins {
		if o == origin {
			return true
		}
	}
	for _, w := range c.allowedWOrigins {
		if w.match(origin) {
			return true
		}
	}
	return false
}

func (c *corsMiddleware) isMethodAllowed(method string) bool {
	if len(c.allowedMethods) == 0 {
		return false
	}
	method = strings.ToUpper(method)
	if method == http.MethodOptions {
		return true
	}
	for _, m := range c.allowedMethods {
		if m == method {
			return true
		}
	}
	return false
}

func (c *corsMiddleware) areHeadersAllowed(requestedHeaders []string) bool {
	if c.allowedHeadersAll || len(requestedHeaders) == 0 {
		return true
	}
	for _, header := range requestedHeaders {
		header = http.CanonicalHeaderKey(header)
		found := false
		for _, h := range c.allowedHeaders {
			if h == header {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

const toLower = 'a' - 'A'

type converter func(string) string

type wildcard struct {
	prefix string
	suffix string
}

func (w wildcard) match(s string) bool {
	return len(s) >= len(w.prefix)+len(w.suffix) && strings.HasPrefix(s, w.prefix) && strings.HasSuffix(s, w.suffix)
}

func convert(s []string, c converter) []string {
	out := make([]string, 0, len(s))
	for _, i := range s {
		out = append(out, c(i))
	}
	return out
}

func parseHeaderList(headerList string) []string {
	l := len(headerList)
	h := make([]byte, 0, l)
	upper := true
	t := 0
	for i := 0; i < l; i++ {
		if headerList[i] == ',' {
			t++
		}
	}
	headers := make([]string, 0, t)
	for i := 0; i < l; i++ {
		b := headerList[i]
		switch {
		case b >= 'a' && b <= 'z':
			if upper {
				h = append(h, b-toLower)
			} else {
				h = append(h, b)
			}
		case b >= 'A' && b <= 'Z':
			if !upper {
				h = append(h, b+toLower)
			} else {
				h = append(h, b)
			}
		case b == '-' || b == '_' || b == '.' || (b >= '0' && b <= '9'):
			h = append(h, b)
		}

		if b == ' ' || b == ',' || i == l-1 {
			if len(h) > 0 {
				headers = append(headers, string(h))
				h = h[:0]
				upper = true
			}
		} else {
			upper = b == '-' || b == '_'
		}
	}
	return headers
}

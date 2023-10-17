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

package cors

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/wildcard"
	"github.com/aacfactory/fns/transports"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type Config struct {
	AllowedOrigins      []string `json:"allowedOrigins"`
	AllowedHeaders      []string `json:"allowedHeaders"`
	ExposedHeaders      []string `json:"exposedHeaders"`
	AllowCredentials    bool     `json:"allowCredentials"`
	MaxAge              int      `json:"maxAge"`
	AllowPrivateNetwork bool     `json:"allowPrivateNetwork"`
}

type builder struct {
}

func (builder *builder) Name() string {
	return "cors"
}

func (builder *builder) Build(options transports.MiddlewareBuilderOptions) (middleware transports.Middleware, err error) {
	config := Config{}
	err = options.Config.As(&config)
	if err != nil {
		err = errors.Warning("fns: build cors middleware failed").WithCause(err)
		return
	}
	allowedOrigins := make([]string, 0, 1)
	allowedWOrigins := make([]*wildcard.Wildcard, 0, 1)
	allowedOriginsAll := false
	if config.AllowedHeaders == nil {
		config.AllowedHeaders = make([]string, 0, 1)
	}
	if len(config.AllowedHeaders) == 0 || config.AllowedHeaders[0] != "*" {
		defaultAllowedHeaders := []string{
			transports.OriginHeaderName, transports.AcceptHeaderName, transports.ContentTypeHeaderName, transports.XRequestedWithHeaderName,
			transports.ConnectionHeaderName, transports.UpgradeHeaderName,
			transports.XForwardedForHeaderName, transports.TrueClientIpHeaderName, transports.XRealIpHeaderName,
			transports.DeviceIpHeaderName, transports.DeviceIdHeaderName,
			transports.RequestIdHeaderName,
			transports.RequestInternalSignatureHeaderName, transports.RequestTimeoutHeaderName, transports.RequestVersionsHeaderName,
			transports.ETagHeaderName, transports.CacheControlHeaderIfNonMatch, transports.ClearSiteDataHeaderName, transports.ResponseRetryAfterHeaderName, transports.SignatureHeaderName,
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
			w := wildcard.New(origin)
			allowedWOrigins = append(allowedWOrigins, w)
		} else {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}
	allowedHeadersAll := false
	allowedHeaders := make([]string, 0, 1)
	allowedHeaders = builder.convert(append(config.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
	for _, h := range config.AllowedHeaders {
		if h == "*" {
			allowedHeadersAll = true
			allowedHeaders = nil
			break
		}
	}

	if config.ExposedHeaders == nil {
		config.ExposedHeaders = make([]string, 0, 1)
	}
	defaultExposedHeaders := []string{
		transports.RequestIdHeaderName, transports.RequestInternalSignatureHeaderName, transports.HandleLatencyHeaderName,
		transports.CacheControlHeaderName, transports.ETagHeaderName, transports.ClearSiteDataHeaderName, transports.ResponseRetryAfterHeaderName, transports.ResponseCacheTTLHeaderName, transports.SignatureHeaderName,
	}
	for _, header := range defaultExposedHeaders {
		if sort.SearchStrings(config.ExposedHeaders, header) < 0 {
			config.ExposedHeaders = append(config.ExposedHeaders, header)
		}
	}

	middleware = &corsMiddleware{
		allowedOrigins:      allowedOrigins,
		allowedWOrigins:     allowedWOrigins,
		allowedOriginsAll:   allowedOriginsAll,
		allowedHeaders:      allowedHeaders,
		allowedHeadersAll:   allowedHeadersAll,
		allowedMethods:      []string{http.MethodGet, http.MethodPost, http.MethodHead},
		exposedHeaders:      builder.convert(config.ExposedHeaders, http.CanonicalHeaderKey),
		maxAge:              config.MaxAge,
		allowCredentials:    config.AllowCredentials,
		allowPrivateNetwork: config.AllowPrivateNetwork,
		handler:             nil,
	}

	return
}

func CORS() transports.MiddlewareBuilder {
	return &builder{}
}

type corsMiddleware struct {
	allowedOrigins      []string
	allowedWOrigins     []*wildcard.Wildcard
	allowedOriginsAll   bool
	allowedHeaders      []string
	allowedHeadersAll   bool
	allowedMethods      []string
	exposedHeaders      []string
	maxAge              int
	allowCredentials    bool
	allowPrivateNetwork bool
	handler             transports.Handler
}

func (builder *builder) convert(s []string, converter func(string) string) []string {
	out := make([]string, 0, len(s))
	for _, i := range s {
		out = append(out, converter(i))
	}
	return out
}

func (c *corsMiddleware) Handler(next transports.Handler) transports.Handler {
	c.handler = next
	return c
}

func (c *corsMiddleware) Handle(w transports.ResponseWriter, r *transports.Request) {
	if bytex.ToString(r.Method()) == http.MethodOptions && r.Header().Get(transports.AccessControlRequestMethodHeaderName) != "" {
		c.handlePreflight(w, r)
		w.SetStatus(http.StatusNoContent)
	} else {
		c.handleActualRequest(w, r)
		c.handler.Handle(w, r)
	}
}

func (c *corsMiddleware) handlePreflight(w transports.ResponseWriter, r *transports.Request) {
	headers := w.Header()
	origin := r.Header().Get(transports.OriginHeaderName)

	if bytex.ToString(r.Method()) != http.MethodOptions {
		return
	}
	headers.Add(transports.VaryHeaderName, transports.OriginHeaderName)
	headers.Add(transports.VaryHeaderName, transports.AccessControlRequestMethodHeaderName)
	headers.Add(transports.VaryHeaderName, transports.AccessControlRequestHeadersHeaderName)
	if c.allowPrivateNetwork {
		headers.Add(transports.VaryHeaderName, transports.AccessControlRequestPrivateNetworkHeaderName)
	}

	if origin == "" {
		return
	}
	if !c.isOriginAllowed(origin) {
		return
	}

	reqMethod := r.Header().Get(transports.AccessControlRequestMethodHeaderName)
	if !c.isMethodAllowed(reqMethod) {
		return
	}
	reqHeaders := c.parseHeaderList(r.Header().Get(transports.AccessControlRequestHeadersHeaderName))
	if !c.areHeadersAllowed(reqHeaders) {
		return
	}
	if c.allowedOriginsAll {
		headers.Set(transports.AccessControlAllowOriginHeaderName, "*")
	} else {
		headers.Set(transports.AccessControlAllowOriginHeaderName, origin)
	}
	headers.Set(transports.AccessControlAllowMethodsHeaderName, strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {
		headers.Set(transports.AccessControlAllowHeadersHeaderName, strings.Join(reqHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set(transports.AccessControlAllowCredentialsHeaderName, "true")
	}
	if c.allowPrivateNetwork && r.Header().Get(transports.AccessControlRequestPrivateNetworkHeaderName) == "true" {
		headers.Set(transports.AccessControlAllowPrivateNetworkHeaderName, "true")
	}
	if c.maxAge > 0 {
		headers.Set(transports.AccessControlMaxAgeHeaderName, strconv.Itoa(c.maxAge))
	}
}

func (c *corsMiddleware) handleActualRequest(w transports.ResponseWriter, r *transports.Request) {
	headers := w.Header()
	origin := r.Header().Get(transports.OriginHeaderName)

	headers.Add(transports.VaryHeaderName, transports.OriginHeaderName)
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
		headers.Set(transports.AccessControlAllowOriginHeaderName, "*")
	} else {
		headers.Set(transports.AccessControlAllowOriginHeaderName, origin)
	}
	if len(c.exposedHeaders) > 0 {
		headers.Set(transports.AccessControlExposeHeadersHeaderName, strings.Join(c.exposedHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set(transports.AccessControlAllowCredentialsHeaderName, "true")
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
		if w.Match(origin) {
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

func (c *corsMiddleware) parseHeaderList(headerList string) []string {
	const (
		toLower = 'a' - 'A'
	)
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

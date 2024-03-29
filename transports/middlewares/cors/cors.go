/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package cors

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/wildcard"
	"github.com/aacfactory/fns/transports"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

func New() transports.Middleware {
	return &corsMiddleware{}
}

type corsMiddleware struct {
	allowedOrigins      [][]byte
	allowedWOrigins     []*wildcard.Wildcard
	allowedOriginsAll   bool
	allowedHeaders      [][]byte
	allowedHeadersAll   bool
	allowedMethods      [][]byte
	exposedHeaders      [][]byte
	maxAge              int
	allowCredentials    bool
	allowPrivateNetwork bool
	preflightVary       [][]byte
	handler             transports.Handler
}

func (c *corsMiddleware) Name() string {
	return "cors"
}

func (c *corsMiddleware) Construct(options transports.MiddlewareOptions) (err error) {
	config := Config{}
	err = options.Config.As(&config)
	if err != nil {
		err = errors.Warning("fns: build cors middleware failed").WithCause(err)
		return
	}
	allowedOrigins := make([][]byte, 0, 1)
	allowedWOrigins := make([]*wildcard.Wildcard, 0, 1)
	allowedOriginsAll := false
	if config.AllowedHeaders == nil {
		config.AllowedHeaders = make([]string, 0, 1)
	}
	if len(config.AllowedHeaders) == 0 || config.AllowedHeaders[0] != "*" {
		defaultAllowedHeaders := []string{
			string(transports.OriginHeaderName), string(transports.AcceptHeaderName), string(transports.ContentTypeHeaderName),
			string(transports.AcceptEncodingHeaderName),
			string(transports.XRequestedWithHeaderName),
			string(transports.ConnectionHeaderName), string(transports.UpgradeHeaderName),
			string(transports.XForwardedForHeaderName), string(transports.TrueClientIpHeaderName), string(transports.XRealIpHeaderName),
			string(transports.DeviceIpHeaderName), string(transports.DeviceIdHeaderName),
			string(transports.RequestIdHeaderName),
			string(transports.RequestTimeoutHeaderName), string(transports.RequestVersionsHeaderName),
			string(transports.CacheControlHeaderIfNonMatch), string(transports.CacheControlHeaderName),
			string(transports.SignatureHeaderName),
		}
		for _, header := range defaultAllowedHeaders {
			if !slices.Contains(config.AllowedHeaders, header) {
				config.AllowedHeaders = append(config.AllowedHeaders, header)
			}
		}
	}
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}
	for _, origin := range config.AllowedOrigins {
		origin = strings.ToLower(origin)
		if origin == "*" {
			allowedOriginsAll = true
			allowedOrigins = nil
			allowedWOrigins = nil
			break
		} else if i := strings.IndexByte(origin, '*'); i >= 0 {
			w := wildcard.New(bytex.FromString(origin))
			allowedWOrigins = append(allowedWOrigins, w)
		} else {
			allowedOrigins = append(allowedOrigins, bytex.FromString(origin))
		}
	}
	allowedHeadersAll := false
	allowedHeaders := make([][]byte, 0, 1)
	for _, header := range config.AllowedHeaders {
		allowedHeaders = append(allowedHeaders, bytex.FromString(header))
	}
	allowedHeaders = convert(allowedHeaders, http.CanonicalHeaderKey)
	for _, h := range config.AllowedHeaders {
		if h == "*" {
			allowedHeadersAll = true
			allowedHeaders = nil
			break
		}
	}

	exposedHeaders := make([][]byte, 0, 1)
	if config.ExposedHeaders == nil {
		config.ExposedHeaders = make([]string, 0, 1)
	}
	defaultExposedHeaders := []string{
		string(transports.VaryHeaderName),
		string(transports.DeviceIdHeaderName),
		string(transports.EndpointIdHeaderName), string(transports.EndpointVersionHeaderName),
		string(transports.ContentEncodingHeaderName),
		string(transports.RequestIdHeaderName), string(transports.HandleLatencyHeaderName),
		string(transports.CacheControlHeaderName), string(transports.ETagHeaderName), string(transports.ClearSiteDataHeaderName), string(transports.AgeHeaderName),
		string(transports.ResponseRetryAfterHeaderName), string(transports.SignatureHeaderName),
		string(transports.DeprecatedHeaderName),
	}
	for _, header := range defaultExposedHeaders {
		if !slices.Contains(config.ExposedHeaders, header) {
			config.ExposedHeaders = append(config.ExposedHeaders, header)
		}
	}
	for _, header := range config.ExposedHeaders {
		exposedHeaders = append(exposedHeaders, bytex.FromString(header))
	}
	exposedHeaders = convert(exposedHeaders, http.CanonicalHeaderKey)

	c.allowedOrigins = allowedOrigins
	c.allowedWOrigins = allowedWOrigins
	c.allowedOriginsAll = allowedOriginsAll
	c.allowedHeaders = allowedHeaders
	c.allowedHeadersAll = allowedHeadersAll
	c.allowedMethods = [][]byte{methodGet, methodPost, methodHead}
	c.exposedHeaders = exposedHeaders
	c.maxAge = config.MaxAge
	c.allowCredentials = config.AllowCredentials
	c.allowPrivateNetwork = config.AllowPrivateNetwork

	if c.allowPrivateNetwork {
		c.preflightVary = [][]byte{[]byte("Origin, Access-Control-Request-Method, Access-Control-Request-Headers, Access-Control-Request-Private-Network")}
	} else {
		c.preflightVary = [][]byte{[]byte("Origin, Access-Control-Request-Method, Access-Control-Request-Headers")}
	}
	return
}

func (c *corsMiddleware) Handler(next transports.Handler) transports.Handler {
	c.handler = next
	return c
}

func (c *corsMiddleware) Close() (err error) {
	return
}

func (c *corsMiddleware) Handle(w transports.ResponseWriter, r transports.Request) {
	if bytes.Equal(r.Method(), methodOptions) && len(r.Header().Get(accessControlRequestMethodHeader)) > 0 {
		c.handlePreflight(w, r)
		w.SetStatus(http.StatusNoContent)
	} else {
		c.handleActualRequest(w, r)
		c.handler.Handle(w, r)
	}
}

func (c *corsMiddleware) handlePreflight(w transports.ResponseWriter, r transports.Request) {
	headers := w.Header()
	origin := r.Header().Get(originHeader)

	if !bytes.Equal(r.Method(), methodOptions) {
		return
	}

	if vary := headers.Get(varyHeader); len(vary) > 0 {
		headers.Add(varyHeader, c.preflightVary[0])
	} else {
		for _, preflightVary := range c.preflightVary {
			headers.Add(varyHeader, preflightVary)
		}
	}

	if len(origin) == 0 {
		return
	}
	if !c.isOriginAllowed(origin) {
		return
	}

	reqMethod := r.Header().Get(accessControlRequestMethodHeader)
	if !c.isMethodAllowed(reqMethod) {
		return
	}
	reqHeadersRaw := r.Header().Values(accessControlRequestHeadersHeader)
	reqHeaders, reqHeadersEdited := parseHeaderList(reqHeadersRaw)
	if !c.areHeadersAllowed(reqHeaders) {
		return
	}
	if c.allowedOriginsAll {
		headers.Set(accessControlAllowOriginHeader, all)
	} else {
		origins := w.Header().Values(originHeader)
		for _, ori := range origins {
			headers.Add(accessControlAllowOriginHeader, ori)
		}
	}
	headers.Set(accessControlAllowMethodsHeader, bytes.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {
		if reqHeadersEdited || len(reqHeaders) != len(reqHeadersRaw) {
			headers.Set(accessControlAllowHeadersHeader, bytes.Join(reqHeaders, joinBytes))
		} else {
			for _, raw := range reqHeadersRaw {
				headers.Add(accessControlAllowHeadersHeader, raw)
			}
		}
	}
	if c.allowCredentials {
		headers.Set(accessControlAllowCredentialsHeader, trueBytes)
	}

	if c.allowPrivateNetwork && bytes.Equal(r.Header().Get(accessControlRequestPrivateNetworkHeader), trueBytes) {
		headers.Set(accessControlAllowPrivateNetworkHeader, trueBytes)
	}

	if c.maxAge > 0 {
		headers.Set(accessControlMaxAgeHeader, bytex.FromString(strconv.Itoa(c.maxAge)))
	}
}

func (c *corsMiddleware) handleActualRequest(w transports.ResponseWriter, r transports.Request) {
	headers := w.Header()
	origin := r.Header().Get(originHeader)

	if len(origin) == 0 {
		return
	}
	if !c.isOriginAllowed(origin) {
		return
	}

	if !c.isMethodAllowed(r.Method()) {
		return
	}
	if c.allowedOriginsAll {
		headers.Set(accessControlAllowOriginHeader, all)
	} else {
		origins := w.Header().Values(originHeader)
		for _, ori := range origins {
			headers.Add(accessControlAllowOriginHeader, ori)
		}
	}
	if len(c.exposedHeaders) > 0 {
		for _, exposedHeader := range c.exposedHeaders {
			headers.Add(accessControlExposeHeadersHeader, exposedHeader)
		}
	}
	if c.allowCredentials {
		headers.Set(accessControlAllowCredentialsHeader, trueBytes)
	}
}

func (c *corsMiddleware) isOriginAllowed(origin []byte) bool {
	if c.allowedOriginsAll {
		return true
	}
	origin = bytes.ToLower(origin)
	for _, o := range c.allowedOrigins {
		if bytes.Equal(o, origin) {
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

func (c *corsMiddleware) isMethodAllowed(method []byte) bool {
	if len(c.allowedMethods) == 0 {
		return false
	}
	ms := bytes.ToUpper(method)
	if bytes.Equal(ms, methodOptions) {
		return true
	}
	for _, m := range c.allowedMethods {
		if bytes.Equal(ms, m) {
			return true
		}
	}
	return false
}

func (c *corsMiddleware) areHeadersAllowed(requestedHeaders [][]byte) bool {
	if c.allowedHeadersAll || len(requestedHeaders) == 0 {
		return true
	}
	for _, header := range requestedHeaders {
		hs := bytex.FromString(http.CanonicalHeaderKey(bytex.ToString(header)))
		found := false
		for _, h := range c.allowedHeaders {
			if bytes.Equal(hs, h) {
				found = true
				break
			}
			if bytes.Index(hs, transports.UserHeaderNamePrefix) == 0 {
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

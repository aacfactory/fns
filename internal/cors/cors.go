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
	"net/http"
	"strconv"
	"strings"
)

type Options struct {
	AllowedOrigins         []string
	AllowOriginFunc        func(origin string) bool
	AllowOriginRequestFunc func(r *http.Request, origin string) bool
	AllowedMethods         []string
	AllowedHeaders         []string
	ExposedHeaders         []string
	MaxAge                 int
	AllowCredentials       bool
	AllowPrivateNetwork    bool
	OptionsPassthrough     bool
	OptionsSuccessStatus   int
}

// Cors http handler
type Cors struct {
	allowedOrigins         []string
	allowedWOrigins        []wildcard
	allowOriginFunc        func(origin string) bool
	allowOriginRequestFunc func(r *http.Request, origin string) bool
	allowedHeaders         []string
	allowedMethods         []string
	exposedHeaders         []string
	maxAge                 int
	allowedOriginsAll      bool
	allowedHeadersAll      bool
	optionsSuccessStatus   int
	allowCredentials       bool
	allowPrivateNetwork    bool
	optionPassthrough      bool
}

func New(options Options) *Cors {
	c := &Cors{
		exposedHeaders:         convert(options.ExposedHeaders, http.CanonicalHeaderKey),
		allowOriginFunc:        options.AllowOriginFunc,
		allowOriginRequestFunc: options.AllowOriginRequestFunc,
		allowCredentials:       options.AllowCredentials,
		allowPrivateNetwork:    options.AllowPrivateNetwork,
		maxAge:                 options.MaxAge,
		optionPassthrough:      options.OptionsPassthrough,
	}

	if len(options.AllowedOrigins) == 0 {
		if options.AllowOriginFunc == nil && options.AllowOriginRequestFunc == nil {
			c.allowedOriginsAll = true
		}
	} else {
		c.allowedOrigins = []string{}
		c.allowedWOrigins = []wildcard{}
		for _, origin := range options.AllowedOrigins {
			origin = strings.ToLower(origin)
			if origin == "*" {
				c.allowedOriginsAll = true
				c.allowedOrigins = nil
				c.allowedWOrigins = nil
				break
			} else if i := strings.IndexByte(origin, '*'); i >= 0 {
				w := wildcard{origin[0:i], origin[i+1:]}
				c.allowedWOrigins = append(c.allowedWOrigins, w)
			} else {
				c.allowedOrigins = append(c.allowedOrigins, origin)
			}
		}
	}

	if len(options.AllowedHeaders) == 0 {
		c.allowedHeaders = []string{"Origin", "Accept", "Content-Type", "X-Requested-With"}
	} else {
		c.allowedHeaders = convert(append(options.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
		for _, h := range options.AllowedHeaders {
			if h == "*" {
				c.allowedHeadersAll = true
				c.allowedHeaders = nil
				break
			}
		}
	}

	if len(options.AllowedMethods) == 0 {
		c.allowedMethods = []string{http.MethodGet, http.MethodPost, http.MethodHead}
	} else {
		c.allowedMethods = convert(options.AllowedMethods, strings.ToUpper)
	}

	if options.OptionsSuccessStatus == 0 {
		c.optionsSuccessStatus = http.StatusNoContent
	} else {
		c.optionsSuccessStatus = options.OptionsSuccessStatus
	}

	return c
}

func Default() *Cors {
	return New(Options{})
}

func AllowAll() *Cors {
	return New(Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	})
}

func (c *Cors) Handle(writer http.ResponseWriter, request *http.Request) (ok bool) {
	if request.Method == http.MethodOptions && request.Header.Get("Access-Control-Request-Method") != "" {
		c.handlePreflight(writer, request)
		if c.optionPassthrough {
			ok = false
		} else {
			writer.WriteHeader(c.optionsSuccessStatus)
			ok = true
		}
	} else {
		c.handleActualRequest(writer, request)
		ok = false
	}
	return
}

func (c *Cors) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			c.handlePreflight(w, r)

			if c.optionPassthrough {
				h.ServeHTTP(w, r)
			} else {
				w.WriteHeader(c.optionsSuccessStatus)
			}
		} else {
			c.handleActualRequest(w, r)
			h.ServeHTTP(w, r)
		}
	})
}

func (c *Cors) HandlerFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
		c.handlePreflight(w, r)

		w.WriteHeader(c.optionsSuccessStatus)
	} else {
		c.handleActualRequest(w, r)
	}
}

func (c *Cors) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
		c.handlePreflight(w, r)

		if c.optionPassthrough {
			next(w, r)
		} else {
			w.WriteHeader(c.optionsSuccessStatus)
		}
	} else {
		c.handleActualRequest(w, r)
		next(w, r)
	}
}

func (c *Cors) handlePreflight(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	origin := r.Header.Get("Origin")

	if r.Method != http.MethodOptions {
		return
	}
	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")

	if origin == "" {
		return
	}
	if !c.isOriginAllowed(r, origin) {
		return
	}

	reqMethod := r.Header.Get("Access-Control-Request-Method")
	if !c.isMethodAllowed(reqMethod) {
		return
	}
	reqHeaders := parseHeaderList(r.Header.Get("Access-Control-Request-Headers"))
	if !c.areHeadersAllowed(reqHeaders) {
		return
	}
	if c.allowedOriginsAll {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", origin)
	}
	headers.Set("Access-Control-Allow-Methods", strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {

		headers.Set("Access-Control-Allow-Headers", strings.Join(reqHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	if c.allowPrivateNetwork && r.Header.Get("Access-Control-Request-Private-Network") == "true" {
		headers.Set("Access-Control-Allow-Private-Network", "true")
	}
	if c.maxAge > 0 {
		headers.Set("Access-Control-Max-Age", strconv.Itoa(c.maxAge))
	}
}

func (c *Cors) handleActualRequest(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	origin := r.Header.Get("Origin")

	headers.Add("Vary", "Origin")
	if origin == "" {
		return
	}
	if !c.isOriginAllowed(r, origin) {
		return
	}

	if !c.isMethodAllowed(r.Method) {
		return
	}
	if c.allowedOriginsAll {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", origin)
	}
	if len(c.exposedHeaders) > 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(c.exposedHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
}

func (c *Cors) OriginAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return c.isOriginAllowed(r, origin)
}

func (c *Cors) isOriginAllowed(r *http.Request, origin string) bool {
	if c.allowOriginRequestFunc != nil {
		return c.allowOriginRequestFunc(r, origin)
	}
	if c.allowOriginFunc != nil {
		return c.allowOriginFunc(origin)
	}
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

func (c *Cors) isMethodAllowed(method string) bool {
	if len(c.allowedMethods) == 0 {
		// If no method allowed, always return false, even for preflight request
		return false
	}
	method = strings.ToUpper(method)
	if method == http.MethodOptions {
		// Always allow preflight requests
		return true
	}
	for _, m := range c.allowedMethods {
		if m == method {
			return true
		}
	}
	return false
}

func (c *Cors) areHeadersAllowed(requestedHeaders []string) bool {
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

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

package fns

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aacfactory/fns/commons"
	"github.com/valyala/fasthttp"
)

const toLower = 'a' - 'A'

var (
	corsAccessOrigin        = []byte("Access-Control-Allow-Origin")
	corsAccessControlHeader = []byte("Access-Control-Request-Method")
	requestOriginHeader     = []byte("Origin")
	requestSecFetchMode     = []byte("Sec-Fetch-Mode")
)

func newCors(config CorsConfig) *cors {
	c := &cors{
		exposedHeaders:   config.ExposedHeaders,
		allowCredentials: config.AllowCredentials,
		maxAge:           config.MaxAge,
	}
	if len(config.AllowedOrigins) == 0 {
		c.allowedOriginsAll = true
	} else {
		c.allowedOrigins = []string{}
		c.allowedWOrigins = []commons.Wildcard{}
		for _, origin := range config.AllowedOrigins {
			origin = strings.ToLower(origin)
			if origin == "*" {
				c.allowedOriginsAll = true
				c.allowedOrigins = nil
				c.allowedWOrigins = nil
				break
			} else if i := strings.IndexByte(origin, '*'); i >= 0 {
				w := commons.Wildcard{Prefix: origin[0:i], Suffix: origin[i+1:]}
				c.allowedWOrigins = append(c.allowedWOrigins, w)
			} else {
				c.allowedOrigins = append(c.allowedOrigins, origin)
			}
		}
	}

	if len(config.AllowedHeaders) == 0 {
		c.allowedHeaders = []string{"Origin", "Accept", "Accept-Encoding", "Authorization", "Content-Type", "X-Requested-With", "X-Forward-For"}
	} else {
		c.allowedHeaders = append(config.AllowedHeaders, "Origin")
		for _, h := range config.AllowedHeaders {
			if h == "*" {
				c.allowedHeadersAll = true
				c.allowedHeaders = nil
				break
			}
		}
	}

	if len(config.AllowedMethods) == 0 {
		c.allowedMethods = []string{"HEAD", "GET", "POST"}
	} else {
		c.allowedMethods = config.AllowedMethods
	}

	return c
}

type cors struct {
	allowedOrigins    []string
	allowedWOrigins   []commons.Wildcard
	allowedHeaders    []string
	allowedMethods    []string
	exposedHeaders    []string
	allowedOriginsAll bool
	allowedHeadersAll bool
	allowCredentials  bool
	maxAge            int
}

func (c *cors) handler(h fasthttp.RequestHandler) (ch fasthttp.RequestHandler) {
	ch = func(ctx *fasthttp.RequestCtx) {
		if ctx.IsOptions() {
			accessControlRequestMethod := string(ctx.Request.Header.PeekBytes(corsAccessControlHeader))
			if accessControlRequestMethod == "" {
				h(ctx)
				c.writeAccessControlAllowOrigin(ctx)
				return
			}
			c.handlePreflight(ctx)
			ctx.SetStatusCode(204)
		} else {
			h(ctx)
			c.writeAccessControlAllowOrigin(ctx)
		}
	}
	return
}
func (c *cors) writeAccessControlAllowOrigin(ctx *fasthttp.RequestCtx) {
	origin := string(ctx.Request.Header.PeekBytes(requestOriginHeader))
	if origin == "" {
		return
	}
	if c.allowedOriginsAll {
		ctx.Response.Header.SetBytesK(corsAccessOrigin, "*")
	} else {
		ctx.Response.Header.SetBytesKV(corsAccessOrigin, ctx.Request.Header.PeekBytes(requestOriginHeader))
	}
}

func (c *cors) handlePreflight(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Add("Vary", "Origin")
	ctx.Response.Header.Add("Vary", "Access-Control-Request-Method")
	ctx.Response.Header.Add("Vary", "Access-Control-Request-Headers")

	origin := string(ctx.Request.Header.PeekBytes(requestOriginHeader))

	if origin == "" {
		return
	}

	if !c.isOriginAllowed(origin) {
		return
	}

	reqMethod := string(ctx.Request.Header.PeekBytes(corsAccessControlHeader))
	if !c.isMethodAllowed(reqMethod) {
		return
	}

	reqHeaders := c.parseHeaderList(string(ctx.Request.Header.Peek("Access-Control-Request-Headers")))
	if !c.areHeadersAllowed(reqHeaders) {
		return
	}
	if c.allowedOriginsAll {
		ctx.Response.Header.SetBytesK(corsAccessOrigin, "*")
	} else {
		ctx.Response.Header.SetBytesK(corsAccessOrigin, origin)
	}

	ctx.Response.Header.Set("Access-Control-Allow-Methods", strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {
		ctx.Response.Header.Set("Access-Control-Allow-Headers", strings.Join(reqHeaders, ", "))
	}
	if c.allowCredentials {
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	}
	if c.maxAge > 0 {
		ctx.Response.Header.Set("Access-Control-Max-Age", strconv.Itoa(c.maxAge))
	}
}

func (c *cors) isOriginAllowed(origin string) bool {
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

func (c *cors) isMethodAllowed(method string) bool {
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

func (c *cors) areHeadersAllowed(requestedHeaders []string) bool {
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

func (c *cors) parseHeaderList(headerList string) []string {
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
				// Flush the found header
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

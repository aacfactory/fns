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

package cachecontrol

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"net/http"
	"strconv"
	"time"
)

var (
	nilBodyETag      = []byte(strconv.FormatUint(xxhash.Sum64([]byte("nil")), 16))
	contextEnableKey = []byte("@fns:cache-control:enable")
)

func NewWithCache(cache Cache) transports.Middleware {
	return &Middleware{
		cache: cache,
	}
}

func New() transports.Middleware {
	return NewWithCache(new(DefaultCache))
}

var (
	getMethod = []byte("GET")
)

// Middleware
// use @cache-control max-age=10 public=true must-revalidate=false proxy-revalidate=false
type Middleware struct {
	log    logs.Logger
	cache  Cache
	enable bool
	maxAge int
}

func (middleware *Middleware) Name() string {
	return "cachecontrol"
}

func (middleware *Middleware) Construct(options transports.MiddlewareOptions) (err error) {
	middleware.log = options.Log
	config := Config{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: construct cache control middleware failed").WithCause(configErr)
		return
	}
	if config.Enable {
		middleware.enable = config.Enable
		middleware.maxAge = config.MaxAge
		if middleware.maxAge < 1 {
			middleware.maxAge = 60
		}
	}
	return
}

func (middleware *Middleware) Handler(next transports.Handler) transports.Handler {
	if middleware.enable {
		return transports.HandlerFunc(func(writer transports.ResponseWriter, request transports.Request) {
			isGet := bytes.Equal(request.Method(), getMethod)
			if !isGet {
				next.Handle(writer, request)
				return
			}
			rch := request.Header().Get(transports.CacheControlHeaderName)
			if bytes.Equal(rch, transports.CacheControlHeaderNoStore) || bytes.Equal(rch, transports.CacheControlHeaderNoCache) {
				next.Handle(writer, request)
				return
			}
			// request key
			var key []byte
			// if-no-match
			if inm := request.Header().Get(transports.CacheControlHeaderIfNonMatch); len(inm) > 0 {
				key = hashRequest(request)
				etag, hasEtag, getErr := middleware.cache.Get(request, key)
				if getErr != nil {
					if middleware.log.WarnEnabled() {
						middleware.log.Warn().
							With("middleware", "cachecontrol").
							Cause(errors.Warning("fns: get etag from cache store failed").WithCause(getErr)).
							Message("get etag from cache store failed")
					}
					next.Handle(writer, request)
					return
				}
				if hasEtag && bytes.Equal(etag, inm) {
					writer.SetStatus(http.StatusNotModified)
					return
				}
			}
			// set enable
			request.SetLocalValue(contextEnableKey, true)
			// next
			next.Handle(writer, request)
			// check response
			cch := writer.Header().Get(transports.CacheControlHeaderName)
			if len(cch) == 0 {
				return
			}
			maxAgeValue := 0
			hasMaxAge := false
			if idx := bytes.Index(cch, maxAge); idx > -1 {
				segment := cch[idx+8:]
				commaIdx := bytes.IndexByte(segment, ',')
				if commaIdx > 0 {
					segment = segment[:commaIdx]
				}
				ma, parseErr := strconv.Atoi(bytex.ToString(segment))
				if parseErr == nil {
					maxAgeValue = ma
					hasMaxAge = true
				} else {
					// remove invalid max-age
					if idx == 0 {
						cch = cch[idx+8+commaIdx+1:]
					} else {
						cch = append(cch[:idx], cch[idx+8+commaIdx+1:]...)
					}
				}
			}
			if !hasMaxAge {
				maxAgeValue = middleware.maxAge
				cch = append(cch, ',', ' ')
				cch = append(cch, maxAge...)
				cch = append(cch, '=')
				cch = append(cch, strconv.Itoa(maxAgeValue)...)
				writer.Header().Set(transports.CacheControlHeaderName, cch)
			}
			if len(key) == 0 {
				key = hashRequest(request)
			}
			// etag
			var etag []byte
			if bodyLen := writer.BodyLen(); bodyLen == 0 {
				etag = nilBodyETag
			} else {
				body := writer.Body()
				etag = bytex.FromString(strconv.FormatUint(xxhash.Sum64(body), 16))
			}
			setErr := middleware.cache.Set(request, key, etag, time.Duration(maxAgeValue)*time.Second)
			if setErr == nil {
				writer.Header().Set(transports.ETagHeaderName, etag)
			} else {
				if middleware.log.WarnEnabled() {
					middleware.log.Warn().
						With("middleware", "cachecontrol").
						Cause(errors.Warning("fns: set etag into cache store failed").WithCause(setErr)).
						Message("set etag into cache store failed")
				}
				// use no-cache
				writer.Header().Set(transports.CacheControlHeaderName, transports.CacheControlHeaderNoCache)
				writer.Header().Set(pragma, transports.CacheControlHeaderNoCache)
			}
		})
	}
	return next
}

func (middleware *Middleware) Close() (err error) {
	middleware.cache.Close()
	return
}

func hashRequest(r transports.Request) (p []byte) {
	b := bytebufferpool.Get()
	// device id
	deviceId := r.Header().Get(transports.DeviceIdHeaderName)
	_, _ = b.Write(deviceId)
	// path
	_, _ = b.Write(r.Path())
	// param
	param := r.Params()
	_, _ = b.Write(param.Encode())
	// authorization
	if authorization := r.Header().Get(transports.AuthorizationHeaderName); len(authorization) > 0 {
		_, _ = b.Write(authorization)
	}
	pp := b.Bytes()
	p = bytex.FromString(strconv.FormatUint(xxhash.Sum64(pp), 16))
	bytebufferpool.Put(b)
	return
}

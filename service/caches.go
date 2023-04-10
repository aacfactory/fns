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
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service/shareds"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/cespare/xxhash/v2"
	"github.com/valyala/bytebufferpool"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cacheControlMiddlewareName = "cachecontrol"
	cacheControlContextKey     = "@fns_cachecontrol"
)

type cacheControlMiddlewareConfig struct {
	TTL string `json:"ttl"`
}

func CacheControlMiddleware() TransportMiddleware {
	return &cacheControlMiddleware{}
}

type cacheControlMiddleware struct {
	log   logs.Logger
	store shareds.Store
	ttl   time.Duration
	pool  sync.Pool
}

func (middleware *cacheControlMiddleware) Name() (name string) {
	name = cacheControlMiddlewareName
	return
}

func (middleware *cacheControlMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.log = options.Log
	middleware.store = options.Shared.Store()
	config := cacheControlMiddlewareConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: build cache control middleware failed").WithCause(configErr)
		return
	}
	if config.TTL != "" {
		middleware.ttl, err = time.ParseDuration(strings.TrimSpace(config.TTL))
		if err != nil {
			err = errors.Warning("fns: build cache control middleware failed").WithCause(errors.Warning("fns: ttl must be time.Duration format")).WithCause(err)
			return
		}
	}
	if middleware.ttl < 1 {
		middleware.ttl = 30 * time.Minute
	}
	middleware.pool = sync.Pool{
		New: func() any {
			return &cacheControl{}
		},
	}
	return
}

func (middleware *cacheControlMiddleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r *transports.Request) {
		// 0. discard upgrade and internal(cache by caller endpoint)
		if r.Header().Get(httpUpgradeHeader) != "" || r.Header().Get(httpRequestInternalHeader) == "true" {
			next.Handle(w, r)
			return
		}
		// 1. check if no match
		ifNonMatch := r.Header().Get(httpCacheControlIfNonMatch)
		if ifNonMatch != "" {
			etag := bytex.FromString(ifNonMatch)
			exist := middleware.existETag(r.Context(), etag)
			if exist {
				w.SetStatus(http.StatusNotModified)
				return
			}
		}
		ctx := r.Context()
		// 2. check cached
		etag := middleware.createETag(r)
		value, etagTTL, status, contentType, contentLength, cached := middleware.getETag(ctx, etag)
		if cached {
			w.Header().Set(httpContentType, contentType)
			w.Header().Set(httpContentLength, contentLength)
			w.Header().Set(httpETagHeader, bytex.ToString(etag))
			w.Header().Set(httpCacheControlHeader, "public, max-age=0")
			w.Header().Set(httpResponseCacheTTL, etagTTL.String())
			w.SetStatus(status)
			_, _ = w.Write(value)
			return
		}
		// 2. check cached request result, key = hash request, value = etag, when exists, get result by etag
		ccx := middleware.pool.Get()
		if ccx == nil {
			ccx = &cacheControl{}
		}
		cc := ccx.(*cacheControl)
		r.WithContext(context.WithValue(ctx, cacheControlContextKey, cc))
		// 3. next
		next.Handle(w, r)
		// 4. cache used
		if !cc.used || w.Hijacked() {
			return
		}
		ttl := cc.ttl
		if ttl < 1 {
			ttl = middleware.ttl
		}
		saveErr := middleware.saveETag(ctx, etag, w.Status(), w.Body(), ttl, w.Header().Get(httpContentType))
		if saveErr != nil {
			w.Header().Set(httpCacheControlHeader, "public, no-store, x-fns-failed")
		} else {
			w.Header().Set(httpETagHeader, bytex.ToString(etag))
			w.Header().Set(httpCacheControlHeader, "public, max-age=0")
			w.Header().Set(httpResponseCacheTTL, ttl.String())
		}
	})
}

func (middleware *cacheControlMiddleware) makeEtagStoreKey(etag []byte) []byte {
	return bytex.FromString(fmt.Sprintf("fns/etags/%s", bytex.ToString(etag)))
}

func (middleware *cacheControlMiddleware) existETag(ctx context.Context, etag []byte) (ok bool) {
	var err error
	ok, err = middleware.store.Exists(ctx, middleware.makeEtagStoreKey(etag))
	if err != nil && middleware.log.ErrorEnabled() {
		middleware.log.Error().Cause(
			errors.Warning("fns: get etag from shared store failed").WithMeta("middleware", middleware.Name()).WithCause(err),
		).Message("fns: get etag from shared store failed")
	}
	return
}

func (middleware *cacheControlMiddleware) getETag(ctx context.Context, etag []byte) (value []byte, ttl time.Duration, status int, ct string, cl string, ok bool) {
	var err error
	value, ok, err = middleware.store.Get(ctx, middleware.makeEtagStoreKey(etag))
	if err != nil {
		if middleware.log.ErrorEnabled() {
			middleware.log.Error().Cause(
				errors.Warning("fns: get etag from shared store failed").WithMeta("middleware", middleware.Name()).WithCause(err),
			).Message("fns: get etag from shared store failed")
		}
		ok = false
		return
	}
	e := cachedEntry{}
	err = json.Unmarshal(value, &e)
	if err != nil {
		if middleware.log.ErrorEnabled() {
			middleware.log.Error().Cause(
				errors.Warning("fns: decode etag value failed").WithMeta("middleware", middleware.Name()).WithCause(err),
			).Message("fns: decode etag value failed")
		}
		ok = false
		return
	}
	value = e.Data
	ttl = e.TTL
	status = e.Status
	ct = e.ContentType
	cl = e.ContentLength
	ok = true
	return
}

func (middleware *cacheControlMiddleware) createETag(r *transports.Request) (etag []byte) {
	buf := bytebufferpool.Get()
	// path
	_, _ = buf.Write(r.Path())
	// body
	if r.Body() != nil {
		_, _ = buf.Write(r.Body())
	}
	etag = bytex.FromString(strconv.FormatUint(xxhash.Sum64(buf.Bytes()), 10))
	bytebufferpool.Put(buf)
	return
}

func (middleware *cacheControlMiddleware) saveETag(ctx context.Context, etag []byte, status int, value []byte, ttl time.Duration, contentType string) (err error) {
	contentLength := len(value)
	if contentType == "" {
		if json.Validate(value) {
			contentType = httpContentTypeJson
		} else {
			l := 512
			if contentLength < 512 {
				l = contentLength
			}
			contentType = http.DetectContentType(value[:l])
		}
	}
	if status == 0 {
		if contentType == httpContentTypeJson {
			obj := json.NewObjectFromBytes(value)
			if obj.Contains("id") && obj.Contains("message") && obj.Contains("stacktrace") {
				status = 555
			} else {
				status = http.StatusOK
			}
		} else {
			status = http.StatusOK
		}
	}
	e := cachedEntry{
		Data:          value,
		TTL:           ttl,
		Status:        status,
		ContentType:   contentType,
		ContentLength: strconv.Itoa(contentLength),
	}
	p, encodeErr := json.Marshal(&e)
	if encodeErr != nil {
		err = errors.Warning("fns: encode etag value failed").WithMeta("middleware", middleware.Name()).WithCause(encodeErr)
		if middleware.log.ErrorEnabled() {
			middleware.log.Error().Cause(err).Message("fns: encode etag value failed")
		}
		return
	}
	setErr := middleware.store.SetWithTTL(ctx, middleware.makeEtagStoreKey(etag), p, ttl)
	if setErr != nil {
		err = errors.Warning("fns: set etag into shared store failed").WithMeta("middleware", middleware.Name()).WithCause(setErr)
		if middleware.log.ErrorEnabled() {
			middleware.log.Error().Cause(err).Message("fns: set etag into shared store failed")
		}
		return
	}
	return
}

func (middleware *cacheControlMiddleware) Close() (err error) {
	return
}

type cachedEntry struct {
	Data          []byte        `json:"data"`
	TTL           time.Duration `json:"ttl"`
	Status        int           `json:"status"`
	ContentType   string        `json:"contentType"`
	ContentLength string        `json:"contentLength"`
}

type cacheControl struct {
	used bool
	ttl  time.Duration
}

func (cc *cacheControl) Enable(ttl time.Duration) {
	cc.used = true
	cc.ttl = ttl
}

// EnableCacheControl
// use `@cache {ttl}` in service function
func EnableCacheControl(ctx context.Context, ttl time.Duration) {
	x := ctx.Value(cacheControlContextKey)
	if x == nil {
		return
	}
	cc, ok := x.(*cacheControl)
	if !ok {
		return
	}
	cc.Enable(ttl)
}

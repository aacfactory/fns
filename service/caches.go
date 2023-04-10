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
	RequestCacheDefaultTTL string `json:"requestCacheDefaultTTL"`
}

func CacheControlMiddleware() TransportMiddleware {
	return &cacheControlMiddleware{}
}

type cacheControlMiddleware struct {
	log  logs.Logger
	tags *ETags
	pool sync.Pool
}

func (middleware *cacheControlMiddleware) Name() (name string) {
	name = cacheControlMiddlewareName
	return
}

func (middleware *cacheControlMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.log = options.Log

	config := cacheControlMiddlewareConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: build cache control middleware failed").WithCause(configErr)
		return
	}
	ttl := time.Duration(0)
	if config.RequestCacheDefaultTTL != "" {
		ttl, err = time.ParseDuration(strings.TrimSpace(config.RequestCacheDefaultTTL))
		if err != nil {
			err = errors.Warning("fns: build cache control middleware failed").WithCause(errors.Warning("fns: requestCacheDefaultTTL must be time.Duration format")).WithCause(err)
			return
		}
	}
	if ttl < 1 {
		ttl = 30 * time.Minute
	}
	middleware.tags = &ETags{
		log:        middleware.log,
		defaultTTL: ttl,
		store:      options.Shared.Caches(),
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
		ctx := r.Context()
		// 1. check if no match
		ifNonMatch := r.Header().Get(httpCacheControlIfNonMatch)
		if ifNonMatch != "" {
			etag := bytex.FromString(ifNonMatch)
			exist := middleware.tags.exist(ctx, etag)
			if exist {
				w.SetStatus(http.StatusNotModified)
				return
			}
		}

		// 2. check cached
		etag := middleware.tags.create(r.Path(), r.Body())
		status, contentType, contentLength, value, etagTTL, cached := middleware.tags.get(ctx, etag)
		if cached {
			w.Header().Set(httpContentType, bytex.ToString(contentType))
			w.Header().Set(httpContentLength, bytex.ToString(contentLength))
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
			ttl = middleware.tags.defaultTTL
		}
		if middleware.tags.save(ctx, etag, w.Status(), bytex.FromString(w.Header().Get(httpContentType)), w.Body(), ttl) {
			w.Header().Set(httpETagHeader, bytex.ToString(etag))
			w.Header().Set(httpCacheControlHeader, "public, max-age=0")
			w.Header().Set(httpResponseCacheTTL, ttl.String())
		} else {
			w.Header().Set(httpCacheControlHeader, "public, no-store, x-fns-failed")
		}
	})
}

func (middleware *cacheControlMiddleware) Close() (err error) {
	return
}

type ETags struct {
	log        logs.Logger
	defaultTTL time.Duration
	store      shareds.Caches
}

func (tags *ETags) enabled() bool {
	return tags.defaultTTL > 0
}

func (tags *ETags) exist(ctx context.Context, etag []byte) (has bool) {
	has = tags.store.Exist(ctx, etag)
	return
}

func (tags *ETags) get(ctx context.Context, etag []byte) (status int, ct []byte, cl []byte, value []byte, ttl time.Duration, has bool) {
	value, has = tags.store.Get(ctx, tags.makeCacheKey(etag))
	if !has {
		return
	}
	e := cachedEntry{}
	err := json.Unmarshal(value, &e)
	if err != nil {
		if tags.log.ErrorEnabled() {
			tags.log.Error().Cause(
				errors.Warning("fns: decode etag value failed").WithCause(err),
			).Message("fns: decode etag value failed")
		}
		has = false
		return
	}
	value = e.Data
	ttl = e.TTL
	status = e.Status
	ct = bytex.FromString(e.ContentType)
	cl = bytex.FromString(e.ContentLength)
	has = true
	return
}

func (tags *ETags) save(ctx context.Context, etag []byte, status int, contentType []byte, value []byte, ttl time.Duration) (ok bool) {
	if ttl < 1 {
		ttl = tags.defaultTTL
	}
	contentLength := len(value)
	ct := bytex.ToString(contentType)
	if ct == "" {
		if json.Validate(value) {
			ct = httpContentTypeJson
		} else {
			l := 512
			if contentLength < 512 {
				l = contentLength
			}
			ct = http.DetectContentType(value[:l])
		}
	}
	if status == 0 {
		if ct == httpContentTypeJson {
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
		ContentType:   ct,
		ContentLength: strconv.Itoa(contentLength),
	}
	p, encodeErr := json.Marshal(&e)
	if encodeErr != nil {
		err := errors.Warning("fns: encode etag value failed").WithCause(encodeErr)
		if tags.log.ErrorEnabled() {
			tags.log.Error().Cause(err).Message("fns: encode etag value failed")
		}
		return
	}
	tags.store.Set(ctx, tags.makeCacheKey(etag), p, ttl)
	ok = true
	return
}

func (tags *ETags) makeCacheKey(etag []byte) []byte {
	return bytex.FromString(fmt.Sprintf("fns/etags/%s", bytex.ToString(etag)))
}

func (tags *ETags) create(path []byte, body []byte) (etag []byte) {
	buf := bytebufferpool.Get()
	// path
	_, _ = buf.Write(path)
	// body
	if body != nil {
		_, _ = buf.Write(body)
	}
	etag = bytex.FromString(strconv.FormatUint(xxhash.Sum64(buf.Bytes()), 10))
	bytebufferpool.Put(buf)
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

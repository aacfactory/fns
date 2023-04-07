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
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/service/shareds"
	"github.com/aacfactory/fns/service/transports"
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
		ifNonMatch := r.Header().Get(httpCacheControlIfNonMatch)
		if ifNonMatch != "" {
			exist := middleware.existETag(r.Context(), ifNonMatch)
			if exist {
				w.SetStatus(http.StatusNotModified)
				return
			}
		}
		ccx := middleware.pool.Get()
		if ccx == nil {
			ccx = &cacheControl{}
		}
		cc := ccx.(*cacheControl)
		r.WithContext(context.WithValue(r.Context(), cacheControlContextKey, cc))
		next.Handle(w, r)
		if !cc.used {
			return
		}
		ttl := cc.ttl
		if ttl < 1 {
			ttl = middleware.ttl
		}
		etag, etagErr := middleware.makeETag(r.Context(), r, ttl)
		if etagErr != nil {
			return
		}
		w.Header().Set(httpETagHeader, etag)
		w.Header().Set(httpCacheControlHeader, "public, max-age=0")
		w.Header().Set(httpResponseCacheTTL, ttl.String())
	})
}

func (middleware *cacheControlMiddleware) makeEtagKey(etag string) string {
	return fmt.Sprintf("fns/etags/%s", etag)
}

func (middleware *cacheControlMiddleware) existETag(ctx context.Context, etag string) (ok bool) {
	var err error
	_, ok, err = middleware.store.Get(ctx, bytex.FromString(middleware.makeEtagKey(etag)))
	if err != nil && middleware.log.ErrorEnabled() {
		middleware.log.Error().Cause(
			errors.Warning("fns: get etag from shared store failed").WithMeta("middleware", middleware.Name()).WithCause(err),
		).Message("fns: get etag from shared store failed")
	}
	return
}

func (middleware *cacheControlMiddleware) makeETag(ctx context.Context, r *transports.Request, ttl time.Duration) (etag string, err error) {
	buf := bytebufferpool.Get()
	// deviceId
	deviceId := r.Header().Get(httpDeviceIdHeader)
	if deviceId == "" {
		deviceIdInQuery := r.Param("deviceId")
		deviceId = bytex.ToString(deviceIdInQuery)
	}
	if deviceId == "" {
		deviceId = uid.UID()
	}
	_, _ = buf.Write(bytex.FromString(deviceId))
	// path
	_, _ = buf.Write(r.Path())
	// body
	if r.Body() != nil {
		_, _ = buf.Write(r.Body())
	}
	etag = strconv.FormatUint(xxhash.Sum64(buf.Bytes()), 10)
	bytebufferpool.Put(buf)

	setErr := middleware.store.SetWithTTL(ctx, bytex.FromString(middleware.makeEtagKey(etag)), bytex.FromString(ttl.String()), ttl)
	if setErr != nil {
		err = errors.Warning("fns: set etag into shared store failed").WithCause(setErr)
		if middleware.log.ErrorEnabled() {
			middleware.log.Error().Cause(
				errors.Warning("fns: set etag into shared store failed").WithMeta("middleware", middleware.Name()).WithCause(setErr),
			).Message("fns: set etag into shared store failed")
		}
		return
	}
	return
}

func (middleware *cacheControlMiddleware) Close() (err error) {
	return
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

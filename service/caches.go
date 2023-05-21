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
	"bytes"
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

var (
	cacheVaryHeaderValue = strings.Join([]string{
		httpRequestIdHeader, httpHandleLatencyHeader, httpSignatureHeader, httpResponseCacheTTL,
	}, ",")
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
		store:      options.Runtime.Shared().Caches(),
	}
	middleware.pool = sync.Pool{
		New: func() any {
			return newCacheControl(middleware.log.With("cachecontrol", "etag"), middleware.tags)
		},
	}
	return
}

func (middleware *cacheControlMiddleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r *transports.Request) {
		// discard upgrade and internal(cache by caller endpoint)
		if r.Header().Get(httpUpgradeHeader) != "" || r.Header().Get(httpRequestInternalHeader) == "true" {
			next.Handle(w, r)
			return
		}
		// set cache control into request context
		ctx := r.Context()
		ccx := middleware.pool.Get()
		if ccx == nil {
			ccx = newCacheControl(middleware.log.With("cachecontrol", "etag"), middleware.tags)
		}
		cc := ccx.(*cacheControl)
		r.WithContext(context.WithValue(ctx, cacheControlContextKey, cc))
		// handle
		var rk []byte
		ccv := bytex.FromString(r.Header().Get(httpCacheControlHeader))
		if bytes.Contains(ccv, bytex.FromString(httpCacheControlNoStore)) || bytes.Contains(ccv, bytex.FromString(httpCacheControlNoCache)) {
			// no-store or no-cache
			next.Handle(w, r)
		} else {
			// load cache
			rk = getOrMakeRequestHash(r.Header(), r.Path(), r.Body())
			tag, cached := middleware.tags.get(ctx, rk)
			if !cached {
				// no cache
				next.Handle(w, r)
			} else {
				// check if no match
				ifNonMatch := r.Header().Get(httpCacheControlIfNonMatch)
				if ifNonMatch != "" {
					if ifNonMatch == tag.Etag {
						w.Header().Set(httpVaryHeader, cacheVaryHeaderValue)
						// not modified
						w.SetStatus(http.StatusNotModified)
						cc.reset()
						middleware.pool.Put(cc)
						return
					}
				}
				// not out of date
				if ttl := tag.Deadline.Sub(time.Now()); ttl > 0 {
					maxAge := int(ttl / time.Second)
					w.Header().Set(httpContentType, tag.ContentType)
					w.Header().Set(httpContentLength, tag.ContentLength)
					w.Header().Set(httpETagHeader, tag.Etag)
					w.Header().Set(httpCacheControlHeader, fmt.Sprintf("public, max-age=%d", maxAge))
					w.Header().Set(httpResponseCacheTTL, strconv.Itoa(int(ttl/time.Second)))
					w.SetStatus(tag.Status)
					_, _ = w.Write(tag.Data)
					cc.reset()
					middleware.pool.Put(cc)
					return
				}
				// out of date
				next.Handle(w, r)
			}
		}
		// discard when hijacked
		if w.Hijacked() {
			cc.reset()
			middleware.pool.Put(cc)
			return
		}
		if rk == nil {
			rk = getOrMakeRequestHash(r.Header(), r.Path(), r.Body())
		}
		// write header
		etag, ttl, enabled := cc.Enabled(rk)
		if enabled {
			maxAge := int(ttl / time.Second)
			w.Header().Set(httpETagHeader, etag)
			w.Header().Set(httpCacheControlHeader, fmt.Sprintf("public, max-age=%d", maxAge))
			w.Header().Set(httpResponseCacheTTL, strconv.Itoa(maxAge))
		} else {
			w.Header().Set(httpCacheControlHeader, "public, no-store")
		}
		cc.reset()
		middleware.pool.Put(cc)
		return
	})
}

func (middleware *cacheControlMiddleware) Close() (err error) {
	return
}

type tagValue struct {
	Etag          string    `json:"etag"`
	Deadline      time.Time `json:"deadline"`
	Status        int       `json:"status"`
	Data          []byte    `json:"data"`
	ContentType   string    `json:"contentType"`
	ContentLength string    `json:"contentLength"`
}

type ETags struct {
	log        logs.Logger
	defaultTTL time.Duration
	store      shareds.Caches
}

func (tags *ETags) get(ctx context.Context, rk []byte) (tag *tagValue, has bool) {
	value, exist := tags.store.Get(ctx, tags.makeCacheKey(rk))
	if !exist {
		return
	}
	tag = &tagValue{}
	err := json.Unmarshal(value, tag)
	if err != nil {
		if tags.log.ErrorEnabled() {
			tags.log.Error().Cause(
				errors.Warning("fns: decode etag value failed").WithCause(err),
			).Message("fns: decode etag value failed")
		}
		has = false
		return
	}
	has = true
	return
}

func (tags *ETags) save(ctx context.Context, rk []byte, value *tagValue, ttl time.Duration) {
	p, encodeErr := json.Marshal(value)
	if encodeErr != nil {
		if tags.log.ErrorEnabled() {
			err := errors.Warning("fns: encode etag value failed").WithCause(encodeErr)
			tags.log.Error().Cause(err).Message("fns: encode etag value failed")
		}
		return
	}
	_, ok := tags.store.Set(ctx, tags.makeCacheKey(rk), p, ttl)
	if !ok {
		if tags.log.ErrorEnabled() {
			err := errors.Warning("fns: save etag")
			tags.log.Error().Cause(err).Message("fns: save etag")
		}
	}
	return
}

func (tags *ETags) remove(ctx context.Context, rk []byte) {
	tags.store.Remove(ctx, tags.makeCacheKey(rk))
	return
}

func (tags *ETags) makeCacheKey(rk []byte) []byte {
	return bytex.FromString(fmt.Sprintf("fns/etags/%s", bytex.ToString(rk)))
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

func newCacheControl(log logs.Logger, tags *ETags) *cacheControl {
	return &cacheControl{
		log:     log,
		locker:  new(sync.Mutex),
		tags:    tags,
		entries: make(map[string]*tagValue),
	}
}

type cacheControl struct {
	log     logs.Logger
	locker  sync.Locker
	tags    *ETags
	entries map[string]*tagValue
}

func (cc *cacheControl) reset() {
	if len(cc.entries) > 0 {
		cc.entries = make(map[string]*tagValue)
	}
	return
}

func (cc *cacheControl) Enabled(rk []byte) (tag string, ttl time.Duration, ok bool) {
	cc.locker.Lock()
	e, has := cc.entries[bytex.ToString(rk)]
	cc.locker.Unlock()
	if !has {
		return
	}
	ttl = e.Deadline.Sub(time.Now())
	if ttl < 1 {
		return
	}
	tag = e.Etag
	ok = true
	return
}

func (cc *cacheControl) Enable(ctx context.Context, rk []byte, result interface{}, ttl time.Duration) {
	cc.locker.Lock()

	if ttl < 1 {
		ttl = cc.tags.defaultTTL
	}

	value := tagValue{
		Etag:          "",
		Data:          nil,
		Deadline:      time.Now().Add(ttl),
		Status:        0,
		ContentType:   httpContentTypeJson,
		ContentLength: "",
	}
	if failed, ok := result.(error); ok {
		cr := errors.Map(failed)
		value.Status = cr.Code()
		value.Data, _ = json.Marshal(cr)
	} else {
		value.Status = http.StatusOK
		if result != nil {
			p, encodeErr := json.Marshal(result)
			if encodeErr != nil {
				if cc.log.WarnEnabled() {
					cc.log.Warn().Cause(errors.Warning("fns: cache control encode result failed").WithCause(encodeErr)).Message("fns: cache control enable failed")
				}
				cc.locker.Unlock()
				return
			}
			value.Data = p
		}
	}
	value.ContentLength = strconv.Itoa(len(value.Data))
	value.Etag = strconv.FormatUint(xxhash.Sum64(value.Data), 10)

	cc.tags.save(ctx, rk, &value, ttl)

	cc.entries[bytex.ToString(rk)] = &value

	cc.locker.Unlock()
	return
}

func (cc *cacheControl) Disable(ctx context.Context, rk []byte) {
	cc.locker.Lock()
	delete(cc.entries, bytex.ToString(rk))
	cc.tags.remove(ctx, rk)
	cc.locker.Unlock()
	return
}

// CacheControl
// use `@cache {ttl}` in service function
func CacheControl(ctx context.Context, result interface{}, ttl time.Duration) {
	x := ctx.Value(cacheControlContextKey)
	if x == nil {
		return
	}
	r, hasRequest := GetRequest(ctx)
	if !hasRequest {
		return
	}
	cc, ok := x.(*cacheControl)
	if !ok {
		return
	}
	cc.Enable(ctx, r.Hash(), result, ttl)
}

func RemoveCacheControl(ctx context.Context, service string, fn string, arg Argument) {
	x := ctx.Value(cacheControlContextKey)
	if x == nil {
		return
	}
	cc, ok := x.(*cacheControl)
	if !ok {
		return
	}
	if arg == nil {
		arg = EmptyArgument()
	}
	body, _ := json.Marshal(arg)
	rk := makeRequestHash(bytes.Join([][]byte{emptyBytes, bytex.FromString(service), bytex.FromString(fn)}, slashBytes), body)
	cc.Disable(ctx, rk)
}

func FetchCacheControl(ctx context.Context, service string, fn string, arg Argument) (etag string, status int, contentType string, contentLength string, deadline time.Time, body []byte, has bool) {
	x := ctx.Value(cacheControlContextKey)
	if x == nil {
		return
	}
	cc, ok := x.(*cacheControl)
	if !ok {
		return
	}
	cc.locker.Lock()
	if arg == nil {
		arg = EmptyArgument()
	}
	reqBody, _ := json.Marshal(arg)
	rk := makeRequestHash(bytes.Join([][]byte{emptyBytes, bytex.FromString(service), bytex.FromString(fn)}, slashBytes), reqBody)
	value, cached := cc.entries[bytex.ToString(rk)]
	if cached {
		etag = value.Etag
		status = value.Status
		contentType = value.ContentType
		contentLength = value.ContentLength
		deadline = value.Deadline
		body = value.Data
		has = true
		cc.locker.Unlock()
		return
	}
	value, cached = cc.tags.get(ctx, rk)
	if cached {
		cc.entries[bytex.ToString(rk)] = value
		etag = value.Etag
		status = value.Status
		contentType = value.ContentType
		contentLength = value.ContentLength
		deadline = value.Deadline
		body = value.Data
		has = true
	}
	cc.locker.Unlock()
	return
}

func cacheControlFetch(ctx context.Context, r Request) (etag string, status int, deadline time.Time, body []byte, has bool) {
	service, fn := r.Fn()
	etag, status, _, _, deadline, body, has = FetchCacheControl(ctx, service, fn, r.Argument())
	return
}

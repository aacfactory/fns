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
	"crypto/sha1"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/mmhash"
	"github.com/dgraph-io/ristretto"
	"github.com/valyala/bytebufferpool"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type CacheControlConfig struct {
	MaxAge  int64  `json:"maxAge"`
	Weak    bool   `json:"weak"`
	MaxCost string `json:"maxCost"`
}

func NewCacheControl(config CacheControlConfig) (cache *CacheControl, err error) {
	maxAge := config.MaxAge
	if maxAge < 1 {
		err = errors.Warning("fns: max age of cache control config must be greater than 1")
		return
	}
	maxCost := uint64(64 * bytex.MEGABYTE)
	if config.MaxCost != "" {
		maxCost, err = bytex.ToBytes(strings.TrimSpace(config.MaxCost))
		if err != nil {
			err = errors.Warning("fns: max cost of cache control config must be bytes format")
			return
		}
	}
	if maxCost < uint64(10*bytex.MEGABYTE) {
		maxCost = uint64(64 * bytex.MEGABYTE)
	}
	tags, createCacheErr := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     int64(maxCost),
		BufferItems: 64,
		Metrics:     false,
	})
	if createCacheErr != nil {
		err = errors.Warning("fns: create cache control failed").WithCause(createCacheErr)
		return
	}
	cache = &CacheControl{
		ETags:  tags,
		MaxAge: maxAge,
		Weak:   config.Weak,
	}
	return
}

type CacheControl struct {
	ETags  *ristretto.Cache
	MaxAge int64
	Weak   bool
}

func (cache *CacheControl) Check(w http.ResponseWriter, header http.Header, u *url.URL, body []byte) (ok bool) {
	etag := header.Get("If-None-Match")
	if etag == "" {
		return
	}
	key := cache.buildKey(u, body)
	item, has := cache.ETags.Get(key)
	if !has {
		return
	}
	cached, isEtag := item.(string)
	if !isEtag {
		return
	}
	if cached == etag {
		w.WriteHeader(http.StatusNotModified)
		ok = true
		return
	}
	return
}

func (cache *CacheControl) Set(w http.ResponseWriter, header http.Header, u *url.URL, body []byte) {
	key := cache.buildKey(u, body)
	age := int64(0)
	if header != nil {
		control := header.Get("Cache-Control")
		if strings.Contains(control, "no-cache") || strings.Contains(control, "no-store") {
			cache.ETags.Del(key)
			return
		}
		age = cache.GetMaxAge(header)
	}
	if age == 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", cache.MaxAge))
	}
	hash := sha1.Sum(body)
	etag := fmt.Sprintf("\"%d-%x\"", len(hash), hash)
	if cache.Weak {
		etag = "W/" + etag
	}
	cache.ETags.SetWithTTL(key, etag, int64(len(etag)), time.Duration(age)*time.Second)
	return
}

func (cache *CacheControl) GetMaxAge(header http.Header) (age int64) {
	control := header.Get("Cache-Control")
	idx := strings.Index(control, "max-age")
	if idx > -1 {
		control = control[idx:]
		end := strings.Index(control, ",")
		if end > -1 {
			control = control[0:end]
		}
		mid := strings.Index(control, "=")
		if mid > -1 && len(control) > mid+1 {
			sec := strings.TrimSpace(control[mid+1:])
			age, _ = strconv.ParseInt(sec, 10, 64)
		}
	}
	return
}

func (cache *CacheControl) buildKey(u *url.URL, body []byte) (key uint64) {
	buf := bytebufferpool.Get()
	_, _ = buf.WriteString(u.String())
	if body != nil && len(body) > 0 {
		_, _ = buf.Write(body)
	}
	key = mmhash.MemHash(buf.Bytes())
	bytebufferpool.Put(buf)
	return
}

func (cache *CacheControl) Close() {
	cache.ETags.Close()
}

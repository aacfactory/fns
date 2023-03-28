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
	"strconv"
	"strings"
	"time"
)

type CacheControlConfig struct {
	Weak    bool   `json:"weak"`
	MaxCost string `json:"maxCost"`
}

func NewCacheControl(config CacheControlConfig) (cache *CacheControl, err error) {
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
		ETags: tags,
		Weak:  config.Weak,
	}
	return
}

type CacheControl struct {
	ETags *ristretto.Cache
	Weak  bool
}

func (cache *CacheControl) Cached(path string, header http.Header, body []byte) (key uint64, ok bool) {
	key = cache.buildKey(path, header, body)
	etag := header.Get(httpCacheControlIfNonMatch)
	if etag == "" {
		return
	}
	item, has := cache.ETags.Get(key)
	if !has {
		return
	}
	cached, isEtag := item.(string)
	if !isEtag {
		return
	}
	if cached == etag {
		ok = true
		return
	}
	return
}

func (cache *CacheControl) MaxAge(header http.Header) (age int64, has bool) {
	if header == nil || len(header) == 0 {
		return
	}
	control := header.Get(httpCacheControlHeader)
	if control == "" ||
		strings.Contains(control, "no-cache") ||
		strings.Contains(control, "no-store") ||
		!strings.Contains(control, "max-age") {
		return
	}
	idx := strings.Index(control, "max-age")
	if idx < 0 {
		return
	}
	control = control[idx:]
	end := strings.Index(control, ",")
	if end > -1 {
		control = control[0:end]
	}
	mid := strings.Index(control, "=")
	if mid < 0 || len(control) <= mid+1 {
		return
	}
	sec := strings.TrimSpace(control[mid+1:])
	age, _ = strconv.ParseInt(sec, 10, 64)
	has = age > 0
	return
}

func (cache *CacheControl) CreateETag(key uint64, age int64, body []byte) (v string) {
	hash := sha1.Sum(body)
	v = fmt.Sprintf("\"%d-%x\"", len(hash), hash)
	if cache.Weak {
		v = "W/" + v
	}
	cache.ETags.SetWithTTL(key, v, int64(len(v)), time.Duration(age)*time.Second)
	return
}

func (cache *CacheControl) buildKey(path string, header http.Header, body []byte) (key uint64) {
	buf := bytebufferpool.Get()
	_, _ = buf.WriteString(path)
	devId := header.Get(httpDeviceIdHeader)
	_, _ = buf.WriteString(devId)
	acceptVersions := header.Values(httpRequestVersionsHeader)
	if acceptVersions != nil && len(acceptVersions) > 0 {
		for _, version := range acceptVersions {
			_, _ = buf.WriteString(version)
		}
	}
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

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

package transports

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"net/textproto"
	"sync"
)

var (
	ContentTypeHeaderName                        = []byte("Content-Type")
	ContentTypeJsonHeaderValue                   = []byte("application/json")
	ContentLengthHeaderName                      = []byte("Content-Length")
	AuthorizationHeaderName                      = []byte("Authorization")
	CookieHeaderName                             = []byte("Cookie")
	ConnectionHeaderName                         = []byte("Connection")
	UpgradeHeaderName                            = []byte("Upgrade")
	CloseHeaderValue                             = []byte("close")
	AcceptEncodingHeaderName                     = []byte("Accept-Encoding")
	ContentEncodingHeaderName                    = []byte("Content-Encoding")
	ClearSiteDataHeaderName                      = []byte("Clear-Site-Data")
	CacheControlHeaderName                       = []byte("Cache-Control")
	AgeHeaderName                                = []byte("Age")
	CacheControlHeaderNoStore                    = []byte("no-store")
	CacheControlHeaderNoCache                    = []byte("no-cache")
	ETagHeaderName                               = []byte("ETag")
	CacheControlHeaderIfNonMatch                 = []byte("If-None-Match")
	VaryHeaderName                               = []byte("Vary")
	OriginHeaderName                             = []byte("Origin")
	AcceptHeaderName                             = []byte("Accept")
	AccessControlRequestMethodHeaderName         = []byte("Access-Control-Request-Method")
	AccessControlRequestHeadersHeaderName        = []byte("Access-Control-Request-Headers")
	AccessControlRequestPrivateNetworkHeaderName = []byte("Access-Control-Request-Private-Network")
	AccessControlAllowOriginHeaderName           = []byte("Access-Control-Allow-Origin")
	AccessControlAllowMethodsHeaderName          = []byte("Access-Control-Allow-Methods")
	AccessControlAllowHeadersHeaderName          = []byte("Access-Control-Allow-Headers")
	AccessControlAllowCredentialsHeaderName      = []byte("Access-Control-Allow-Credentials")
	AccessControlAllowPrivateNetworkHeaderName   = []byte("Access-Control-Allow-Private-Network")
	AccessControlMaxAgeHeaderName                = []byte("Access-Control-Max-Age")
	AccessControlExposeHeadersHeaderName         = []byte("Access-Control-Expose-Headers")
	XRequestedWithHeaderName                     = []byte("X-Requested-With")
	TrueClientIpHeaderName                       = []byte("True-Client-Ip")
	XRealIpHeaderName                            = []byte("X-Real-IP")
	XForwardedForHeaderName                      = []byte("X-Forwarded-For")
	RequestIdHeaderName                          = []byte("X-Fns-Request-Id")
	SignatureHeaderName                          = []byte("X-Fns-Signature")
	EndpointIdHeaderName                         = []byte("X-Fns-Endpoint-Id")
	EndpointVersionHeaderName                    = []byte("X-Fns-Endpoint-Version")
	RequestTimeoutHeaderName                     = []byte("X-Fns-Request-Timeout")
	RequestVersionsHeaderName                    = []byte("X-Fns-Request-Version")
	HandleLatencyHeaderName                      = []byte("X-Fns-Handle-Latency")
	DeviceIdHeaderName                           = []byte("X-Fns-Device-Id")
	DeviceIpHeaderName                           = []byte("X-Fns-Device-Ip")
	ResponseRetryAfterHeaderName                 = []byte("Retry-After")
	ResponseTimingAllowOriginHeaderName          = []byte("Timing-Allow-Origin")
	ResponseXFrameOptionsSameOriginHeaderName    = []byte("SAMEORIGIN")
	UserHeaderNamePrefix                         = []byte("XU-")
)

type Header interface {
	Add(key []byte, value []byte)
	Set(key []byte, value []byte)
	Get(key []byte) []byte
	Del(key []byte)
	Values(key []byte) [][]byte
	Foreach(fn func(key []byte, values [][]byte))
	Len() int
	Reset()
}

var (
	headerPool = sync.Pool{}
)

func AcquireHeader() Header {
	cached := headerPool.Get()
	if cached == nil {
		return NewHeader()
	}
	return cached.(Header)
}

func ReleaseHeader(h Header) {
	h.Reset()
	headerPool.Put(h)
}

func NewHeader() Header {
	hh := make(defaultHeader, 0, 1)
	return &hh
}

type headerEntry struct {
	name  []byte
	value [][]byte
}

type defaultHeader []headerEntry

func (h *defaultHeader) Add(key []byte, value []byte) {
	hh := *h
	key = bytex.FromString(textproto.CanonicalMIMEHeaderKey(bytex.ToString(key)))
	for _, entry := range hh {
		if bytes.Equal(entry.name, key) {
			entry.value = append(entry.value, value)
			return
		}
	}
	hh = append(hh, headerEntry{
		name:  key,
		value: [][]byte{value},
	})
	*h = hh
}

func (h *defaultHeader) Set(key []byte, value []byte) {
	hh := *h
	key = bytex.FromString(textproto.CanonicalMIMEHeaderKey(bytex.ToString(key)))
	for _, entry := range hh {
		if bytes.Equal(entry.name, key) {
			entry.value = [][]byte{value}
			return
		}
	}
	hh = append(hh, headerEntry{
		name:  key,
		value: [][]byte{value},
	})
	*h = hh
}

func (h *defaultHeader) Get(key []byte) []byte {
	hh := *h
	key = bytex.FromString(textproto.CanonicalMIMEHeaderKey(bytex.ToString(key)))
	for _, entry := range hh {
		if bytes.Equal(entry.name, key) {
			return entry.value[0]
		}
	}
	return nil
}

func (h *defaultHeader) Del(key []byte) {
	hh := *h
	key = bytex.FromString(textproto.CanonicalMIMEHeaderKey(bytex.ToString(key)))
	n := -1
	for i, entry := range hh {
		if bytes.Equal(entry.name, key) {
			n = i
			break
		}
	}
	if n == -1 {
		return
	}
	hh = append(hh[:n], hh[n+1:]...)
	*h = hh
}

func (h *defaultHeader) Values(key []byte) [][]byte {
	hh := *h
	key = bytex.FromString(textproto.CanonicalMIMEHeaderKey(bytex.ToString(key)))
	for _, entry := range hh {
		if bytes.Equal(entry.name, key) {
			return entry.value
		}
	}
	return nil
}

func (h *defaultHeader) Len() int {
	return len(*h)
}

func (h *defaultHeader) Foreach(fn func(key []byte, values [][]byte)) {
	hh := *h
	for _, entry := range hh {
		fn(entry.name, entry.value)
	}
}

func (h *defaultHeader) Reset() {
	hh := *h
	hh = hh[:0]
	*h = hh
}

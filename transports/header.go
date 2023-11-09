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

package transports

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"net/textproto"
	"sync"
)

const (
	ContentTypeHeaderName                        = "Content-Type"
	ContentTypeJsonHeaderValue                   = "application/json"
	ContentLengthHeaderName                      = "Content-Length"
	AuthorizationHeaderName                      = "Authorization"
	CookieHeaderName                             = "Cookie"
	ConnectionHeaderName                         = "Connection"
	UpgradeHeaderName                            = "Upgrade"
	CloseHeaderValue                             = "close"
	ClearSiteDataHeaderName                      = "Clear-Site-Data"
	CacheControlHeaderName                       = "Cache-Control"
	AgeHeaderName                                = "Age"
	CacheControlHeaderEnabled                    = "public, max-age=0"
	CacheControlHeaderNoStore                    = "no-store"
	CacheControlHeaderNoCache                    = "no-cache"
	ETagHeaderName                               = "ETag"
	CacheControlHeaderIfNonMatch                 = "If-None-Match"
	VaryHeaderName                               = "Vary"
	OriginHeaderName                             = "Origin"
	AcceptHeaderName                             = "Accept"
	AccessControlRequestMethodHeaderName         = "Access-Control-Request-Method"
	AccessControlRequestHeadersHeaderName        = "Access-Control-Request-Headers"
	AccessControlRequestPrivateNetworkHeaderName = "Access-Control-Request-Private-Network"
	AccessControlAllowOriginHeaderName           = "Access-Control-Allow-Origin"
	AccessControlAllowMethodsHeaderName          = "Access-Control-Allow-Methods"
	AccessControlAllowHeadersHeaderName          = "Access-Control-Allow-Headers"
	AccessControlAllowCredentialsHeaderName      = "Access-Control-Allow-Credentials"
	AccessControlAllowPrivateNetworkHeaderName   = "Access-Control-Allow-Private-Network"
	AccessControlMaxAgeHeaderName                = "Access-Control-Max-Age"
	AccessControlExposeHeadersHeaderName         = "Access-Control-Expose-Headers"
	XRequestedWithHeaderName                     = "X-Requested-With"
	TrueClientIpHeaderName                       = "True-Client-Ip"
	XRealIpHeaderName                            = "X-Real-IP"
	XForwardedForHeaderName                      = "X-Forwarded-For"
	RequestIdHeaderName                          = "X-Fns-Request-Id"
	SignatureHeaderName                          = "X-Fns-Signature"
	EndpointIdHeaderName                         = "X-Fns-Endpoint-Id"
	EndpointVersionHeaderName                    = "X-Fns-Endpoint-Version"
	RequestTimeoutHeaderName                     = "X-Fns-Request-Timeout"
	RequestVersionsHeaderName                    = "X-Fns-Request-Version"
	HandleLatencyHeaderName                      = "X-Fns-Handle-Latency"
	DeviceIdHeaderName                           = "X-Fns-Device-Id"
	DeviceIpHeaderName                           = "X-Fns-Device-Ip"
	ResponseRetryAfterHeaderName                 = "Retry-After"
	ResponseTimingAllowOriginHeaderName          = "Timing-Allow-Origin"
	ResponseXFrameOptionsHeaderName              = "X-Frame-Options"
	ResponseXFrameOptionsSameOriginHeaderName    = "SAMEORIGIN"
	UserHeaderNamePrefix                         = "XU-"
)

type Header interface {
	Add(key []byte, value []byte)
	Set(key []byte, value []byte)
	Get(key []byte) []byte
	Del(key []byte)
	Values(key []byte) [][]byte
	Foreach(fn func(key []byte, values [][]byte))
	Reset()
}

var (
	headerPool = sync.Pool{}
)

func AcquireHeader() Header {
	cached := headerPool.Get()
	if cached == nil {
		return newHeader()
	}
	return cached.(Header)
}

func ReleaseHeader(h Header) {
	h.Reset()
	headerPool.Put(h)
}

func newHeader() Header {
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

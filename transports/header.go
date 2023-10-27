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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"net/http"
	"net/textproto"
)

const (
	ContentTypeHeaderName                        = "Content-Type"
	ContentTypeJsonHeaderValue                   = "application/json"
	ContentLengthHeaderName                      = "Content-Length"
	AuthorizationHeaderName                      = "Authorization"
	ConnectionHeaderName                         = "Connection"
	UpgradeHeaderName                            = "Upgrade"
	CloseHeaderValue                             = "close"
	ClearSiteDataHeaderName                      = "Clear-Site-Data"
	CacheControlHeaderName                       = "Cache-Control"
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
	RequestInternalSignatureHeaderName           = "X-Fns-Request-Internal-Signature"
	RequestInternalHeaderName                    = "X-Fns-Request-Internal"
	RequestTimeoutHeaderName                     = "X-Fns-Request-Timeout"
	RequestVersionsHeaderName                    = "X-Fns-Request-Version"
	HandleLatencyHeaderName                      = "X-Fns-Handle-Latency"
	DeviceIdHeaderName                           = "X-Fns-Device-Id"
	DeviceIpHeaderName                           = "X-Fns-Device-Ip"
	DevModeHeaderName                            = "X-Fns-Dev-Mode"
	ResponseRetryAfterHeaderName                 = "Retry-After"
	ResponseCacheTTLHeaderName                   = "X-Fns-Cache-TTL"
	ResponseTimingAllowOriginHeaderName          = "Timing-Allow-Origin"
	ResponseXFrameOptionsHeaderName              = "X-Frame-Options"
	ResponseXFrameOptionsSameOriginHeaderName    = "SAMEORIGIN"
	RequestHashHeaderHeaderName                  = "X-Fns-Request-Hash"
)

type Header interface {
	Add(key []byte, value []byte)
	Set(key []byte, value []byte)
	Get(key []byte) []byte
	Del(key []byte)
	Values(key []byte) [][]byte
	Foreach(fn func(key []byte, values [][]byte))
}

type RequestHeader interface {
	Header
	AcceptedVersions() (intervals versions.Intervals)
	DeviceId() (id []byte)
	DeviceIp() (ip []byte)
	Authorization() (authorization []byte)
	Signature() (signature []byte)
}

func NewHeader() Header {
	return make(httpHeader)
}

func WrapHttpHeader(h http.Header) Header {
	return httpHeader(h)
}

type httpHeader map[string][]string

func (h httpHeader) Add(key []byte, value []byte) {
	textproto.MIMEHeader(h).Add(bytex.ToString(key), bytex.ToString(value))
}

func (h httpHeader) Set(key []byte, value []byte) {
	textproto.MIMEHeader(h).Set(bytex.ToString(key), bytex.ToString(value))
}

func (h httpHeader) Get(key []byte) []byte {
	return bytex.FromString(textproto.MIMEHeader(h).Get(bytex.ToString(key)))
}

func (h httpHeader) Del(key []byte) {
	textproto.MIMEHeader(h).Del(bytex.ToString(key))
}

func (h httpHeader) Values(key []byte) [][]byte {
	vv := textproto.MIMEHeader(h).Values(bytex.ToString(key))
	if len(vv) == 0 {
		return nil
	}
	values := make([][]byte, 0, 1)
	for _, v := range vv {
		values = append(values, bytex.FromString(v))
	}
	return values
}

func (h httpHeader) Foreach(fn func(key []byte, values [][]byte)) {
	if fn == nil {
		return
	}
	for key, values := range h {
		vv := make([][]byte, 0, 1)
		for _, value := range values {
			vv = append(vv, bytex.FromString(value))
		}
		fn(bytex.FromString(key), vv)
	}
}

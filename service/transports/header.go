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
	"net/http"
	"net/textproto"
	"strings"
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

type Header http.Header

func (h Header) Add(key, value string) {
	textproto.MIMEHeader(h).Add(key, value)
}

func (h Header) Set(key, value string) {
	textproto.MIMEHeader(h).Set(key, value)
}

func (h Header) Get(key string) string {
	return textproto.MIMEHeader(h).Get(key)
}

func (h Header) Values(key string) []string {
	return textproto.MIMEHeader(h).Values(key)
}

func (h Header) Del(key string) {
	textproto.MIMEHeader(h).Del(key)
}

func (h Header) Authorization() string {
	return textproto.MIMEHeader(h).Get(AuthorizationHeaderName)
}

func (h Header) Connection() string {
	return textproto.MIMEHeader(h).Get(ConnectionHeaderName)
}

func (h Header) IsConnectionClosed() bool {
	return textproto.MIMEHeader(h).Get(ConnectionHeaderName) == CloseHeaderValue
}

func (h Header) SetConnectionClose() {
	textproto.MIMEHeader(h).Set(ConnectionHeaderName, CloseHeaderValue)
}

func (h Header) Upgrade() string {
	return textproto.MIMEHeader(h).Get(UpgradeHeaderName)
}

func (h Header) ClearSiteData(scopes ...string) {
	if scopes == nil || len(scopes) == 0 {
		textproto.MIMEHeader(h).Set(ClearSiteDataHeaderName, "*")
	} else {
		textproto.MIMEHeader(h).Set(ClearSiteDataHeaderName, `"`+strings.Join(scopes, `", "`)+`"`)
	}
}

func (h Header) Clone() Header {
	if h == nil {
		return nil
	}
	nv := 0
	for _, vv := range h {
		nv += len(vv)
	}
	sv := make([]string, nv)
	h2 := make(Header, len(h))
	for k, vv := range h {
		if vv == nil {
			h2[k] = nil
			continue
		}
		n := copy(sv, vv)
		h2[k] = sv[:n:n]
		sv = sv[n:]
	}
	return h2
}

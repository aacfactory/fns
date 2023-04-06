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
)

const (
	contentTypeHeaderName      = "Content-Type"
	contentTypeJsonHeaderValue = "application/json"
	contentLengthHeaderName    = "Content-Length"
	authorizationHeaderName    = "Authorization"
	connectionHeaderName       = "Connection"
	upgradeHeaderName          = "Upgrade"
	closeHeaderValue           = "close"
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
	return textproto.MIMEHeader(h).Get(authorizationHeaderName)
}

func (h Header) Connection() string {
	return textproto.MIMEHeader(h).Get(connectionHeaderName)
}

func (h Header) IsConnectionClosed() bool {
	return textproto.MIMEHeader(h).Get(connectionHeaderName) == closeHeaderValue
}

func (h Header) SetConnectionClose() {
	textproto.MIMEHeader(h).Set(connectionHeaderName, closeHeaderValue)
}

func (h Header) Upgrade() string {
	return textproto.MIMEHeader(h).Get(upgradeHeaderName)
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

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

package standard

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"net/http"
	"net/textproto"
)

func NewHeader() transports.Header {
	return make(httpHeader)
}

func WrapHttpHeader(h http.Header) transports.Header {
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

func (h httpHeader) Len() int {
	return len(h)
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

func (h httpHeader) Reset() {
	clear(h)
}

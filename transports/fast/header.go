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

package fast

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/valyala/fasthttp"
)

type RequestHeader struct {
	ctx *fasthttp.RequestCtx
}

func (h RequestHeader) Add(key []byte, value []byte) {
	h.ctx.Request.Header.AddBytesKV(key, value)
}

func (h RequestHeader) Set(key []byte, value []byte) {
	h.ctx.Request.Header.SetBytesKV(key, value)
}

func (h RequestHeader) Get(key []byte) []byte {
	return h.ctx.Request.Header.PeekBytes(key)
}

func (h RequestHeader) Del(key []byte) {
	h.ctx.Request.Header.DelBytes(key)
}

func (h RequestHeader) Values(key []byte) [][]byte {
	return h.ctx.Request.Header.PeekAll(bytex.ToString(key))
}

func (h RequestHeader) Foreach(fn func(key []byte, values [][]byte)) {
	if fn == nil {
		return
	}
	keys := make([][]byte, 0)
	h.ctx.Request.Header.VisitAll(func(key, value []byte) {
		keys = append(keys, key)
	})
	for _, key := range keys {
		fn(key, h.Values(key))
	}
}

func (h RequestHeader) Len() int {
	return h.ctx.Request.Header.Len()
}

func (h RequestHeader) Reset() {
	h.ctx.Request.Header.Reset()
}

type ResponseHeader struct {
	*fasthttp.ResponseHeader
}

func (h *ResponseHeader) Add(key []byte, value []byte) {
	h.ResponseHeader.AddBytesKV(key, value)
}

func (h *ResponseHeader) Set(key []byte, value []byte) {
	h.ResponseHeader.SetBytesKV(key, value)
}

func (h *ResponseHeader) Get(key []byte) []byte {
	return h.ResponseHeader.PeekBytes(key)
}

func (h *ResponseHeader) Del(key []byte) {
	h.ResponseHeader.DelBytes(key)
}

func (h *ResponseHeader) Values(key []byte) [][]byte {
	return h.ResponseHeader.PeekAll(bytex.ToString(key))
}

func (h *ResponseHeader) Foreach(fn func(key []byte, values [][]byte)) {
	if fn == nil {
		return
	}
	keys := make([][]byte, 0)
	h.VisitAll(func(key, value []byte) {
		keys = append(keys, key)
	})
	for _, key := range keys {
		fn(key, h.Values(key))
	}
}

func (h *ResponseHeader) Len() int {
	return h.ResponseHeader.Len()
}

func (h *ResponseHeader) Reset() {
	h.ResponseHeader.Reset()
}

package fast

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/valyala/fasthttp"
)

type RequestHeader struct {
	*fasthttp.RequestHeader
}

func (h RequestHeader) Add(key []byte, value []byte) {
	h.RequestHeader.AddBytesKV(key, value)
}

func (h RequestHeader) Set(key []byte, value []byte) {
	h.RequestHeader.SetBytesKV(key, value)
}

func (h RequestHeader) Get(key []byte) []byte {
	return h.RequestHeader.PeekBytes(key)
}

func (h RequestHeader) Del(key []byte) {
	h.RequestHeader.DelBytes(key)
}

func (h RequestHeader) Values(key []byte) [][]byte {
	return h.RequestHeader.PeekAll(bytex.ToString(key))
}

func (h RequestHeader) Foreach(fn func(key []byte, values [][]byte)) {
	if fn == nil {
		return
	}
	keys := h.PeekKeys()
	if len(keys) == 0 {
		return
	}
	for _, key := range keys {
		fn(key, h.RequestHeader.PeekAll(bytex.ToString(key)))
	}
}

type ResponseHeader struct {
	*fasthttp.ResponseHeader
}

func (h ResponseHeader) Add(key []byte, value []byte) {
	h.ResponseHeader.AddBytesKV(key, value)
}

func (h ResponseHeader) Set(key []byte, value []byte) {
	h.ResponseHeader.SetBytesKV(key, value)
}

func (h ResponseHeader) Get(key []byte) []byte {
	return h.ResponseHeader.PeekBytes(key)
}

func (h ResponseHeader) Del(key []byte) {
	h.ResponseHeader.DelBytes(key)
}

func (h ResponseHeader) Values(key []byte) [][]byte {
	return h.ResponseHeader.PeekAll(bytex.ToString(key))
}

func (h ResponseHeader) Foreach(fn func(key []byte, values [][]byte)) {
	if fn == nil {
		return
	}
	keys := h.PeekKeys()
	if len(keys) == 0 {
		return
	}
	for _, key := range keys {
		fn(key, h.ResponseHeader.PeekAll(bytex.ToString(key)))
	}
}

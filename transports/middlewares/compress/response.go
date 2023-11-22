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

package compress

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"sync"
)

var (
	strApplicationSlash = []byte("application/")
	strImageSVG         = []byte("image/svg")
	strImageIcon        = []byte("image/x-icon")
	strFontSlash        = []byte("font/")
	strMultipartSlash   = []byte("multipart/")
	strTextSlash        = []byte("text/")
)

const (
	minCompressLen = 200
)

var (
	responseWriterPool = sync.Pool{}
)

func acquire(w transports.ResponseWriter, c Compressor) *ResponseWriter {
	var cr *ResponseWriter
	v := responseWriterPool.Get()
	if v == nil {
		cr = &ResponseWriter{}
	} else {
		cr = v.(*ResponseWriter)
	}
	cr.ResponseWriter = w
	cr.compressor = c
	return cr
}

func release(w *ResponseWriter) {
	w.compressor = nil
	w.ResponseWriter = nil
	responseWriterPool.Put(w)
}

type ResponseWriter struct {
	transports.ResponseWriter
	compressor Compressor
}

func (w *ResponseWriter) canCompress() bool {
	contentType := w.Header().Get(transports.ContentTypeHeaderName)
	return bytes.HasPrefix(contentType, strTextSlash) ||
		bytes.HasPrefix(contentType, strApplicationSlash) ||
		bytes.HasPrefix(contentType, strImageSVG) ||
		bytes.HasPrefix(contentType, strImageIcon) ||
		bytes.HasPrefix(contentType, strFontSlash) ||
		bytes.HasPrefix(contentType, strMultipartSlash)
}

func (w *ResponseWriter) setCompressHeader() {
	w.Header().Set(transports.ContentEncodingHeaderName, bytex.FromString(w.compressor.Name()))
	vary := w.Header().Get(transports.VaryHeaderName)
	if len(vary) == 0 {
		vary = transports.AcceptEncodingHeaderName
	} else {
		vary = append(vary, ',', ' ')
		vary = append(vary, transports.AcceptEncodingHeaderName...)
	}
	w.Header().Set(transports.VaryHeaderName, vary)
}

func (w *ResponseWriter) Succeed(v interface{}) {
	if v == nil {
		return
	}
	p, encodeErr := json.Marshal(v)
	if encodeErr != nil {
		w.Failed(errors.Warning("fns: transport write succeed result failed").WithCause(encodeErr))
		return
	}
	if len(p) < minCompressLen {
		w.ResponseWriter.Succeed(p)
		return
	}
	compressed, compressErr := w.compressor.Compress(p)
	if compressErr != nil {
		w.ResponseWriter.Succeed(v)
		return
	}
	w.SetStatus(200)
	w.Header().Set(transports.ContentTypeHeaderName, transports.ContentTypeJsonHeaderValue)
	w.setCompressHeader()
	_, _ = w.ResponseWriter.Write(compressed)
}

func (w *ResponseWriter) Failed(cause error) {
	if cause == nil {
		w.ResponseWriter.Failed(cause)
		return
	}
	err := errors.Map(cause)
	body, encodeErr := json.Marshal(err)
	if encodeErr != nil {
		w.SetStatus(555)
		w.Header().Set(transports.ContentTypeHeaderName, transports.ContentTypeJsonHeaderValue)
		body = []byte(`{"message": "fns: transport write failed result failed"}`)
		_, _ = w.ResponseWriter.Write(body)
		return
	}
	if len(body) < minCompressLen {
		w.SetStatus(err.Code())
		w.Header().Set(transports.ContentTypeHeaderName, transports.ContentTypeJsonHeaderValue)
		if bodyLen := len(body); bodyLen > 0 {
			_, _ = w.ResponseWriter.Write(body)
		}
		return
	}
	compressed, compressErr := w.compressor.Compress(body)
	if compressErr != nil {
		w.ResponseWriter.Failed(cause)
		return
	}
	w.SetStatus(err.Code())
	w.Header().Set(transports.ContentTypeHeaderName, transports.ContentTypeJsonHeaderValue)
	w.setCompressHeader()
	_, _ = w.ResponseWriter.Write(compressed)
}

func (w *ResponseWriter) Write(body []byte) (int, error) {
	if len(body) < minCompressLen {
		return w.ResponseWriter.Write(body)
	}
	if w.canCompress() {
		compressed, compressErr := w.compressor.Compress(body)
		if compressErr != nil {
			return w.ResponseWriter.Write(body)
		}
		w.setCompressHeader()
		return w.ResponseWriter.Write(compressed)
	}
	return w.ResponseWriter.Write(body)
}

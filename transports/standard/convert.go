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
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"io"
	"net/http"
	"net/url"
)

func ConvertHttpHandlerFunc(h http.HandlerFunc) transports.Handler {
	return ConvertHttpHandler(h)
}

func ConvertHttpHandler(h http.Handler) transports.Handler {
	return transports.HandlerFunc(func(writer transports.ResponseWriter, request transports.Request) {
		var r http.Request
		if err := ConvertRequest(request, &r, true); err != nil {
			writer.Failed(errors.Warning("fns: convert to http.Request failed").WithCause(err))
			return
		}
		w := netHTTPResponseWriter{w: writer}
		h.ServeHTTP(&w, r.WithContext(request))
		writer.SetStatus(w.StatusCode())
		haveContentType := false
		for k, vv := range w.Header() {
			if k == bytex.ToString(transports.ContentTypeHeaderName) {
				haveContentType = true
			}

			for _, v := range vv {
				writer.Header().Add(bytex.FromString(k), bytex.FromString(v))
			}
		}
		if !haveContentType {
			l := 512
			b := writer.Body()
			if len(b) < 512 {
				l = len(b)
			}
			writer.Header().Set(transports.ContentTypeHeaderName, bytex.FromString(http.DetectContentType(b[:l])))
		}
	})
}

func ConvertRequest(ctx transports.Request, r *http.Request, forServer bool) error {
	body, bodyErr := ctx.Body()
	if bodyErr != nil {
		return bodyErr
	}
	strRequestURI := bytex.ToString(ctx.RequestURI())

	rURL, err := url.ParseRequestURI(strRequestURI)
	if err != nil {
		return err
	}

	r.Method = bytex.ToString(ctx.Method())
	r.Proto = bytex.ToString(ctx.Proto())
	if r.Proto == "HTTP/2" {
		r.ProtoMajor = 2
	} else {
		r.ProtoMajor = 1
	}
	r.ProtoMinor = 1
	r.ContentLength = int64(len(body))
	r.RemoteAddr = bytex.ToString(ctx.RemoteAddr())
	r.Host = bytex.ToString(ctx.Host())
	r.TLS = ctx.TLSConnectionState()
	r.Body = io.NopCloser(bytes.NewReader(body))
	r.URL = rURL

	if forServer {
		r.RequestURI = strRequestURI
	}

	if r.Header == nil {
		r.Header = make(http.Header)
	} else if len(r.Header) > 0 {
		for k := range r.Header {
			delete(r.Header, k)
		}
	}

	ctx.Header().Foreach(func(k []byte, vv [][]byte) {
		sk := bytex.ToString(k)
		for _, v := range vv {
			sv := bytex.ToString(v)
			switch sk {
			case "Transfer-Encoding":
				r.TransferEncoding = append(r.TransferEncoding, sv)
			default:
				r.Header.Add(sk, sv)
			}
		}
	})

	return nil
}

type netHTTPResponseWriter struct {
	statusCode int
	h          http.Header
	w          io.Writer
}

func (w *netHTTPResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func (w *netHTTPResponseWriter) Header() http.Header {
	if w.h == nil {
		w.h = make(http.Header)
	}
	return w.h
}

func (w *netHTTPResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *netHTTPResponseWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

func (w *netHTTPResponseWriter) Flush() {}

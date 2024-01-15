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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"github.com/valyala/fasthttp"
	"slices"
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

func New() transports.Middleware {
	return &Middleware{}
}

type Middleware struct {
	log        logs.Logger
	enable     bool
	compressor Compressor
	gzip       *GzipCompressor
	deflate    *DeflateCompressor
	brotli     *BrotliCompressor
}

func (middle *Middleware) Name() string {
	return "compress"
}

func (middle *Middleware) Construct(options transports.MiddlewareOptions) error {
	middle.log = options.Log
	config := Config{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		return errors.Warning("fns: construct compress middleware failed").WithCause(configErr)
	}
	if !config.Enable {
		return nil
	}
	// gzip
	gzipLevel := config.GzipLevel
	if !slices.Contains([]int{fasthttp.CompressBestSpeed, fasthttp.CompressBestCompression, fasthttp.CompressDefaultCompression, fasthttp.CompressHuffmanOnly}, gzipLevel) {
		gzipLevel = fasthttp.CompressDefaultCompression
	}
	middle.gzip = &GzipCompressor{
		level: gzipLevel,
	}
	// deflate
	deflateLevel := config.DeflateLevel
	if !slices.Contains([]int{fasthttp.CompressBestSpeed, fasthttp.CompressBestCompression, fasthttp.CompressDefaultCompression, fasthttp.CompressHuffmanOnly}, deflateLevel) {
		deflateLevel = fasthttp.CompressDefaultCompression
	}
	middle.deflate = &DeflateCompressor{
		level: deflateLevel,
	}
	// brotli
	brotliLevel := config.BrotliLevel
	if !slices.Contains([]int{fasthttp.CompressBrotliBestSpeed, fasthttp.CompressBrotliBestCompression, fasthttp.CompressBrotliDefaultCompression}, brotliLevel) {
		brotliLevel = fasthttp.CompressBrotliDefaultCompression
	}
	middle.brotli = &BrotliCompressor{
		level: brotliLevel,
	}
	switch config.Default {
	case BrotliName:
		middle.compressor = middle.brotli
		break
	case DeflateName:
		middle.compressor = middle.deflate
		break
	case GzipName:
		middle.compressor = middle.gzip
		break
	default:
		middle.compressor = middle.deflate
		break
	}
	middle.enable = true
	return nil
}

func (middle *Middleware) Handler(next transports.Handler) transports.Handler {
	if middle.enable {
		return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
			next.Handle(w, r)
			if w.BodyLen() < minCompressLen {
				return
			}
			contentType := w.Header().Get(transports.ContentTypeHeaderName)
			canCompress := bytes.HasPrefix(contentType, strTextSlash) ||
				bytes.HasPrefix(contentType, strApplicationSlash) ||
				bytes.HasPrefix(contentType, strImageSVG) ||
				bytes.HasPrefix(contentType, strImageIcon) ||
				bytes.HasPrefix(contentType, strFontSlash) ||
				bytes.HasPrefix(contentType, strMultipartSlash)
			if !canCompress {
				return
			}
			kind := getKind(r)
			var c Compressor = nil
			fmt.Println(kind.String())
			switch kind {
			case Any, Default:
				c = middle.compressor
				break
			case Gzip:
				c = middle.gzip
				break
			case Deflate:
				c = middle.deflate
				break
			case Brotli:
				c = middle.brotli
				break
			default:
				break
			}
			if c == nil {
				return
			}
			body := w.Body()
			compressed, compressErr := c.Compress(body)
			if compressErr != nil {
				if middle.log.WarnEnabled() {
					middle.log.Warn().Cause(compressErr).With("compress", kind.String()).Message("fns: compress response body failed")
				}
				return
			}
			// header
			w.Header().Set(transports.ContentEncodingHeaderName, bytex.FromString(c.Name()))
			w.Header().Add(transports.VaryHeaderName, transports.AcceptEncodingHeaderName)
			// body
			w.ResetBody()
			_, _ = w.Write(compressed)
		})
	}
	return next
}

func (middle *Middleware) Close() (err error) {
	return
}

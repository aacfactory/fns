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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"github.com/valyala/fasthttp"
	"slices"
)

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
	switch config.Default {
	case BrotliName:
		middle.compressor = middle.brotli
		break
	case DeflateName:
		middle.compressor = middle.deflate
		break
	default:
		middle.compressor = middle.gzip
		break
	}
	return nil
}

func (middle *Middleware) Handler(next transports.Handler) transports.Handler {
	if middle.enable {
		return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
			kind := getKind(r)
			switch kind {
			case Any, Default:
				cw := acquire(w, middle.compressor)
				next.Handle(cw, r)
				release(cw)
				break
			case Gzip:
				cw := acquire(w, middle.gzip)
				next.Handle(cw, r)
				release(cw)
				break
			case Deflate:
				cw := acquire(w, middle.deflate)
				next.Handle(cw, r)
				release(cw)
				break
			case Brotli:
				cw := acquire(w, middle.brotli)
				next.Handle(cw, r)
				release(cw)
				break
			default:
				next.Handle(w, r)
				break
			}
		})
	}
	return next
}

func (middle *Middleware) Close() (err error) {
	return
}

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

import "github.com/valyala/fasthttp"

type Compressor interface {
	Name() string
	Compress(p []byte) (out []byte, err error)
}

type GzipCompressor struct {
	level int
}

func (g *GzipCompressor) Name() string {
	return GzipName
}

func (g *GzipCompressor) Compress(p []byte) (out []byte, err error) {
	out = make([]byte, 0, 1)
	fasthttp.AppendGzipBytesLevel(out, p, g.level)
	return
}

type DeflateCompressor struct {
	level int
}

func (d *DeflateCompressor) Name() string {
	return DeflateName
}

func (d *DeflateCompressor) Compress(p []byte) (out []byte, err error) {
	out = make([]byte, 0, 1)
	fasthttp.AppendDeflateBytesLevel(out, p, d.level)
	return
}

type BrotliCompressor struct {
	level int
}

func (b *BrotliCompressor) Name() string {
	return BrotliName
}

func (b *BrotliCompressor) Compress(p []byte) (out []byte, err error) {
	out = make([]byte, 0, 1)
	fasthttp.AppendBrotliBytesLevel(out, p, b.level)
	return
}

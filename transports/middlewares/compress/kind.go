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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
)

const (
	AnyName     = "*"
	DefaultName = "compress"
	GzipName    = "gzip"
	DeflateName = "deflate"
	BrotliName  = "br"
)

const (
	No Kind = iota
	Any
	Default
	Gzip
	Deflate
	Brotli
)

type Kind int

func (kind Kind) String() string {
	switch kind {
	case Default:
		return DefaultName
	case Deflate:
		return DeflateName
	case Gzip:
		return GzipName
	case Brotli:
		return BrotliName
	default:
		return ""
	}
}

func getKind(r transports.Request) Kind {
	accepts := transports.GetAcceptEncodings(r.Header())
	acceptsLen := len(accepts)
	if acceptsLen == 0 {
		return No
	}
	for i := acceptsLen - 1; i >= 0; i-- {
		accept := accepts[i]
		switch bytex.ToString(accept.Name) {
		case DefaultName:
			return Default
		case GzipName:
			return Gzip
		case DeflateName:
			return Deflate
		case BrotliName:
			return Brotli
		case AnyName:
			return Any
		default:
			break
		}
	}
	return No
}

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
	"github.com/valyala/fasthttp"
)

func DecodeResponse(header transports.Header, body []byte) (p []byte, err error) {
	contentEncoding := bytex.ToString(header.Get(transports.ContentEncodingHeaderName))
	switch contentEncoding {
	case DefaultName, GzipName:
		p, err = fasthttp.AppendGunzipBytes(p, body)
		break
	case DeflateName:
		p, err = fasthttp.AppendInflateBytes(p, body)
		break
	case BrotliName:
		p, err = fasthttp.AppendUnbrotliBytes(p, body)
		break
	default:
		p = body
		break
	}
	return
}

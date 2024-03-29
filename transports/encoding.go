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

package transports

import (
	"bytes"
	"github.com/aacfactory/avro"
	"github.com/aacfactory/json"
)

type Marshal func(v any) (p []byte, err error)

func GetMarshaler(ct []byte) (v Marshal, contentType []byte) {
	if len(ct) == 0 {
		v = json.Marshal
		contentType = ContentTypeJsonHeaderValue
		return
	}
	if bytes.Equal(ct, ContentTypeAvroHeaderValue) {
		v = avro.Marshal
		contentType = ContentTypeAvroHeaderValue
		return
	}
	v = json.Marshal
	contentType = ContentTypeJsonHeaderValue
	return
}

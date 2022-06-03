/*
 * Copyright 2021 Wang Min Xiang
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
 */

package cluster

import (
	"encoding/binary"
	"github.com/aacfactory/fns/internal/secret"
	"github.com/valyala/bytebufferpool"
)

func decodeRequestBody(body []byte) (p []byte, ok bool) {
	head := body[0:8]
	signatureLen := binary.BigEndian.Uint64(head)
	signature := body[8 : 8+signatureLen]
	p = body[16+signatureLen:]
	ok = secret.Verify(p, signature)
	return
}

func encodeRequestBody(body []byte) (p []byte) {
	signature := secret.Sign(body)
	head := make([]byte, 8)
	binary.BigEndian.PutUint64(head, uint64(len(signature)))
	buf := bytebufferpool.Get()
	_, _ = buf.Write(head)
	_, _ = buf.Write(signature)
	_, _ = buf.Write(body)
	p = buf.Bytes()
	buf.Reset()
	bytebufferpool.Put(buf)
	return
}

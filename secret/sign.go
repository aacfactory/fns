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

package secret

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func Sign(src []byte, key []byte) (signed []byte) {
	h := hmac.New(sha256.New, key)
	dst := make([]byte, 0, len(src))
	copy(dst, src)
	dst = append(src, '.')
	signature := base64.URLEncoding.EncodeToString(h.Sum(src)[len(src):])
	signed = append(dst, []byte(signature)...)
	return
}

func Verify(signed []byte, key []byte) (ok bool) {
	idx := bytes.LastIndexByte(signed, '.')
	if idx < 1 {
		return
	}
	src := signed[:idx]
	fmt.Println(string(src))
	hashed, hashedErr := base64.URLEncoding.DecodeString(string(signed[idx+1:]))
	if hashedErr != nil {
		return
	}
	target := append(src, hashed...)
	h := hmac.New(sha256.New, key)
	tmp := h.Sum(src)
	ok = hmac.Equal(target, tmp)
	return
}

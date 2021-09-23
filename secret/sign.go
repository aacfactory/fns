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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

func Sign(target []byte, key []byte) (signature []byte) {
	h := hmac.New(sha256.New, key)
	signature = []byte(base64.URLEncoding.EncodeToString(h.Sum(target)))
	return
}

func Verify(target []byte, signature []byte, key []byte) (ok bool) {
	hashed, hashedErr := base64.URLEncoding.DecodeString(string(signature))
	if hashedErr != nil {
		return
	}
	h := hmac.New(sha256.New, key)
	tmp := h.Sum(target)
	ok = hmac.Equal(tmp, hashed)
	return
}

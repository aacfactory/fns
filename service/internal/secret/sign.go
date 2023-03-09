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
	"encoding/hex"
	"github.com/cespare/xxhash/v2"
	"hash"
	"sync"
)

const DefaultSignerKey = "+-fns"

func NewSigner(key []byte) (s *Signer) {
	s = &Signer{
		key:  key,
		pool: sync.Pool{},
	}
	s.pool.New = func() any {
		return hmac.New(func() hash.Hash {
			return xxhash.New()
		}, s.key)
	}
	return
}

type Signer struct {
	key  []byte
	pool sync.Pool
}

func (s *Signer) acquire() (h hash.Hash) {
	v := s.pool.Get()
	if v != nil {
		h = v.(hash.Hash)
		return
	}
	h = hmac.New(func() hash.Hash {
		return xxhash.New()
	}, s.key)
	return
}

func (s *Signer) release(h hash.Hash) {
	h.Reset()
	s.pool.Put(h)
	return
}

func (s *Signer) Sign(target []byte) (signature []byte) {
	h := s.acquire()
	h.Write(target)
	p := h.Sum(nil)
	s.release(h)
	signature = make([]byte, hex.EncodedLen(len(p)))
	hex.Encode(signature, p)
	return
}

func (s *Signer) Verify(target []byte, signature []byte) (ok bool) {
	n, err := hex.Decode(signature, signature)
	if err != nil {
		return
	}
	signature = signature[0:n]
	h := s.acquire()
	h.Write(target)
	p := h.Sum(nil)
	s.release(h)
	ok = hmac.Equal(p, signature)
	return
}

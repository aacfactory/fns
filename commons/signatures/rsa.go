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

package signatures

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"hash"
	"sync"
)

func RSA(pub *rsa.PublicKey, pri *rsa.PrivateKey) Signer {
	return &rsaSigner{
		pub: pub,
		pri: pri,
		pool: sync.Pool{
			New: func() any {
				return sha256.New()
			},
		},
	}
}

type rsaSigner struct {
	pub  *rsa.PublicKey
	pri  *rsa.PrivateKey
	pool sync.Pool
}

func (s *rsaSigner) acquire() (h hash.Hash) {
	v := s.pool.Get()
	if v != nil {
		h = v.(hash.Hash)
		return
	}
	h = sha256.New()
	return
}

func (s *rsaSigner) release(h hash.Hash) {
	h.Reset()
	s.pool.Put(h)
	return
}

func (s *rsaSigner) Sign(target []byte) (signature []byte) {
	h := s.acquire()
	h.Write(target)
	hashed := h.Sum(nil)
	signature, _ = rsa.SignPKCS1v15(rand.Reader, s.pri, crypto.SHA256, hashed)
	s.release(h)
	return
}

func (s *rsaSigner) Verify(target []byte, signature []byte) (ok bool) {
	h := s.acquire()
	h.Write(target)
	hashed := h.Sum(nil)
	ok = rsa.VerifyPKCS1v15(s.pub, crypto.SHA256, hashed, signature) == nil
	s.release(h)
	return
}

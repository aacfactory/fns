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
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"github.com/aacfactory/errors"
	"hash"
	"sync"
)

func ECDSA(keyPEM []byte, hf HashFunc) (Signature, error) {
	block, _ := pem.Decode(keyPEM)
	privateKey, priErr := x509.ParsePKCS8PrivateKey(block.Bytes)
	if priErr != nil {
		return nil, errors.Warning("fns: create ecdsa signer failed").WithCause(errors.Warning("parse private key failed")).WithCause(priErr)
	}
	key, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.Warning("fns: create ecdsa signer failed").WithCause(errors.Warning("private is not ecdsa"))
	}
	return &ecdsaSigner{
		pub: &key.PublicKey,
		pri: key,
		pool: sync.Pool{
			New: func() any {
				return hf()
			},
		},
	}, nil
}

type ecdsaSigner struct {
	pub  *ecdsa.PublicKey
	pri  *ecdsa.PrivateKey
	pool sync.Pool
}

func (s *ecdsaSigner) acquire() (h hash.Hash) {
	v := s.pool.Get()
	if v != nil {
		h = v.(hash.Hash)
		return
	}
	return
}

func (s *ecdsaSigner) release(h hash.Hash) {
	h.Reset()
	s.pool.Put(h)
	return
}

func (s *ecdsaSigner) Sign(target []byte) (signature []byte) {
	h := s.acquire()
	h.Write(target)
	hashed := h.Sum(nil)
	signature, _ = ecdsa.SignASN1(rand.Reader, s.pri, hashed)
	s.release(h)
	return
}

func (s *ecdsaSigner) Verify(target []byte, signature []byte) (ok bool) {
	h := s.acquire()
	h.Write(target)
	hashed := h.Sum(nil)
	ok = ecdsa.VerifyASN1(s.pub, hashed, signature)
	s.release(h)
	return
}

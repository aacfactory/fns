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
	"encoding/base64"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"hash"
	"sync"
)

func ECC(pubBase64 []byte, priBase64 []byte, hf HashFunc) (Signer, error) {
	pub, pubDecodeErr := base64.StdEncoding.DecodeString(bytex.ToString(pubBase64))
	if pubDecodeErr != nil {
		return nil, errors.Warning("fns: create ecc signer failed").WithCause(errors.Warning("parse public key failed")).WithCause(pubDecodeErr)
	}
	publicKeyInterface, pubErr := x509.ParsePKIXPublicKey(pub)
	if pubErr != nil {
		return nil, errors.Warning("fns: create ecc signer failed").WithCause(errors.Warning("parse public key failed")).WithCause(pubErr)
	}
	publicKey := publicKeyInterface.(*ecdsa.PublicKey)

	pri, priDecodeErr := base64.StdEncoding.DecodeString(bytex.ToString(priBase64))
	if priDecodeErr != nil {
		return nil, errors.Warning("fns: create ecc signer failed").WithCause(errors.Warning("parse private key failed")).WithCause(priDecodeErr)
	}
	privateKey, priErr := x509.ParseECPrivateKey(pri)
	if priErr != nil {
		return nil, errors.Warning("fns: create ecc signer failed").WithCause(errors.Warning("parse private key failed")).WithCause(priErr)
	}
	return &eccSigner{
		pub: publicKey,
		pri: privateKey,
		pool: sync.Pool{
			New: func() any {
				return hf()
			},
		},
	}, nil
}

type eccSigner struct {
	pub  *ecdsa.PublicKey
	pri  *ecdsa.PrivateKey
	pool sync.Pool
}

func (s *eccSigner) acquire() (h hash.Hash) {
	v := s.pool.Get()
	if v != nil {
		h = v.(hash.Hash)
		return
	}
	return
}

func (s *eccSigner) release(h hash.Hash) {
	h.Reset()
	s.pool.Put(h)
	return
}

func (s *eccSigner) Sign(target []byte) (signature []byte) {
	h := s.acquire()
	h.Write(target)
	hashed := h.Sum(nil)
	signature, _ = ecdsa.SignASN1(rand.Reader, s.pri, hashed)
	s.release(h)
	return
}

func (s *eccSigner) Verify(target []byte, signature []byte) (ok bool) {
	h := s.acquire()
	h.Write(target)
	hashed := h.Sum(nil)
	ok = ecdsa.VerifyASN1(s.pub, hashed, signature)
	s.release(h)
	return
}

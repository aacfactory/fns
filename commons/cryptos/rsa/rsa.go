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

package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/aacfactory/errors"
)

func New(pubPEM []byte, priPEM []byte) (v *RSA, err error) {
	pubBlock, _ := pem.Decode(pubPEM)
	if pubBlock == nil {
		err = errors.Warning("rsa: public pem is invalid format")
		return
	}
	publicKeyInterface, parsePubErr := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if parsePubErr != nil {
		err = errors.Warning("rsa: parse public pem failed").WithCause(parsePubErr)
		return
	}
	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		err = errors.Warning("rsa: the kind of key is not a rsa.PublicKey")
		return
	}

	priBlock, _ := pem.Decode(priPEM)
	if priBlock == nil {
		err = errors.Warning("rsa: private pem is invalid format")
		return
	}

	// x509 parse
	privateKey, parsePrivateErr := x509.ParsePKCS1PrivateKey(priBlock.Bytes)
	if parsePrivateErr != nil {
		err = errors.Warning("rsa: the kind of key is not a rsa.Private").WithCause(parsePrivateErr)
		return
	}

	v = &RSA{
		public:  publicKey,
		private: privateKey,
	}

	return
}

type RSA struct {
	public  *rsa.PublicKey
	private *rsa.PrivateKey
}

func (r *RSA) Key() (public *rsa.PublicKey, private *rsa.PrivateKey) {
	public, private = r.public, r.private
	return
}

func (r *RSA) Encrypt(plain []byte) (encrypted []byte, err error) {
	encrypted, err = rsa.EncryptPKCS1v15(rand.Reader, r.public, plain)
	if err != nil {
		err = errors.Warning("rsa: encrypt failed").WithCause(err)
		return
	}
	return
}

func (r *RSA) Decrypt(encrypted []byte) (plain []byte, err error) {
	plain, err = rsa.DecryptPKCS1v15(rand.Reader, r.private, encrypted)
	if err != nil {
		err = errors.Warning("rsa: decrypt failed").WithCause(err)
		return
	}
	return
}

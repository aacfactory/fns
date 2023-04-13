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

package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/cryptos/ciphers"
)

func NewEBC(key []byte, padding int) (v *AES, err error) {
	block, parseErr := aes.NewCipher(key)
	if parseErr != nil {
		err = errors.Warning("aes: parse key failed")
		return
	}
	v = &AES{
		key:     block,
		mode:    ciphers.EBC,
		padding: padding,
	}
	return
}

func NewCBC(key []byte, iv []byte, padding int) (v *AES, err error) {
	block, parseErr := aes.NewCipher(key)
	if parseErr != nil {
		err = errors.Warning("aes: parse key failed")
		return
	}
	v = &AES{
		key:     block,
		mode:    ciphers.CBC,
		padding: padding,
		iv:      iv,
	}
	return
}

type AES struct {
	key     cipher.Block
	mode    int
	padding int
	iv      []byte
}

func (a *AES) Encrypt(plain []byte) (encrypted []byte, err error) {
	switch a.mode {
	case ciphers.CBC:
		encrypted, err = ciphers.CBCEncrypt(a.key, plain, a.iv, a.padding)
	case ciphers.EBC:
		encrypted, err = ciphers.ECBEncrypt(a.key, plain, a.padding)
		break
	default:
		err = errors.Warning("aes: unsupported mode")
		break
	}
	if err != nil {
		err = errors.Warning("aes: encrypt failed").WithCause(err)
	}
	return
}

func (a *AES) Decrypt(encrypted []byte) (plain []byte, err error) {
	switch a.mode {
	case ciphers.CBC:
		plain, err = ciphers.CBCDecrypt(a.key, encrypted, a.iv, a.padding)
	case ciphers.EBC:
		plain, err = ciphers.ECBDecrypt(a.key, encrypted, a.padding)
		break
	default:
		err = errors.Warning("aes: unsupported mode")
		break
	}
	if err != nil {
		err = errors.Warning("aes: decrypt failed").WithCause(err)
	}
	return
}

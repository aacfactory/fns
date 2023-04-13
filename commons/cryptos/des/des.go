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

package des

import (
	"crypto/cipher"
	"crypto/des"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/cryptos/ciphers"
)

func NewEBC(key []byte, padding int) (v *DES, err error) {
	block, parseErr := des.NewCipher(key)
	if parseErr != nil {
		err = errors.Warning("des: parse key failed")
		return
	}
	v = &DES{
		key:     block,
		mode:    ciphers.EBC,
		padding: padding,
	}
	return
}

func NewCBC(key []byte, iv []byte, padding int) (v *DES, err error) {
	block, parseErr := des.NewCipher(key)
	if parseErr != nil {
		err = errors.Warning("des: parse key failed")
		return
	}
	v = &DES{
		key:     block,
		mode:    ciphers.CBC,
		padding: padding,
		iv:      iv,
	}
	return
}

type DES struct {
	key     cipher.Block
	mode    int
	padding int
	iv      []byte
}

func (d *DES) Encrypt(plain []byte) (encrypted []byte, err error) {
	switch d.mode {
	case ciphers.CBC:
		encrypted, err = ciphers.CBCEncrypt(d.key, plain, d.iv, d.padding)
	case ciphers.EBC:
		encrypted, err = ciphers.ECBEncrypt(d.key, plain, d.padding)
		break
	default:
		err = errors.Warning("des: unsupported mode")
		break
	}
	if err != nil {
		err = errors.Warning("des: encrypt failed").WithCause(err)
	}
	return
}

func (d *DES) Decrypt(encrypted []byte) (plain []byte, err error) {
	switch d.mode {
	case ciphers.CBC:
		plain, err = ciphers.CBCDecrypt(d.key, encrypted, d.iv, d.padding)
	case ciphers.EBC:
		plain, err = ciphers.ECBDecrypt(d.key, encrypted, d.padding)
		break
	default:
		err = errors.Warning("des: unsupported mode")
		break
	}
	if err != nil {
		err = errors.Warning("des: decrypt failed").WithCause(err)
	}
	return
}

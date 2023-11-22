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

package ciphers

import (
	"crypto/cipher"
	"github.com/aacfactory/errors"
)

func ECBEncrypt(block cipher.Block, plain []byte, padding int) (encrypted []byte, err error) {
	blockSize := block.BlockSize()
	plain = Padding(padding, plain, block.BlockSize())
	plainLen := len(plain)
	encrypted = make([]byte, plainLen)
	if plainLen%blockSize != 0 {
		err = errors.Warning("ebc: input not full blocks")
		return
	}
	p := encrypted[:]
	for len(plain) > 0 {
		block.Encrypt(p, plain[:blockSize])
		plain = plain[blockSize:]
		p = p[blockSize:]
	}
	return
}

func ECBDecrypt(block cipher.Block, encrypted []byte, padding int) (plain []byte, err error) {
	encryptedLen := len(encrypted)
	pad := make([]byte, encryptedLen)
	blockSize := block.BlockSize()
	if encryptedLen%blockSize != 0 {
		err = errors.Warning("ebc: input not full blocks").WithCause(err)
		return
	}
	p := pad[:]
	for len(encrypted) > 0 {
		block.Decrypt(p, encrypted[:blockSize])
		encrypted = encrypted[blockSize:]
		p = p[blockSize:]
	}
	plain, err = UnPadding(padding, pad)
	if err != nil {
		err = errors.Warning("ebc: unPadding failed").WithCause(err)
		return
	}
	return
}

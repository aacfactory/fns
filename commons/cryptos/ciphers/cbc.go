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

func CBCEncrypt(block cipher.Block, plain, iv []byte, padding int) (encrypted []byte, err error) {
	blockSize := block.BlockSize()
	plain = Padding(padding, plain, blockSize)
	encrypted = make([]byte, len(plain))
	if len(iv) != block.BlockSize() {
		err = errors.Warning("cbc: iv length must equal block size")
		return
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, plain)
	return
}

func CBCDecrypt(block cipher.Block, encrypted, iv []byte, padding int) (plain []byte, err error) {
	pad := make([]byte, len(encrypted))
	if len(iv) != block.BlockSize() {
		err = errors.Warning("cbc: IV length must equal block size")
		return
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(pad, encrypted)
	plain, err = UnPadding(padding, pad)
	return
}

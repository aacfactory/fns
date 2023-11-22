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
	"bytes"
	"github.com/aacfactory/errors"
)

var (
	ErrUnPadding = errors.Warning("cbc: unPadding error")
)

const (
	ZEROS = iota
	PKCS5
	PKCS7
)

func Padding(padding int, src []byte, blockSize int) []byte {
	switch padding {
	case PKCS5:
		src = pkcs5Padding(src, blockSize)
	case PKCS7:
		src = pkcs7Padding(src, blockSize)
	case ZEROS:
		src = zerosPadding(src, blockSize)
	}
	return src
}

func UnPadding(padding int, src []byte) ([]byte, error) {
	switch padding {
	case PKCS5:
		return pkcs5UnPadding(src)
	case PKCS7:
		return pkcs7UnPadding(src)
	case ZEROS:
		return zerosUnPadding(src)
	}
	return src, nil
}

func pkcs5Padding(src []byte, blockSize int) []byte {
	return pkcs7Padding(src, blockSize)
}

func pkcs5UnPadding(src []byte) ([]byte, error) {
	return pkcs7UnPadding(src)
}

func pkcs7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	paddings := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, paddings...)
}

func pkcs7UnPadding(src []byte) ([]byte, error) {
	length := len(src)
	if length == 0 {
		return src, ErrUnPadding
	}
	unPadding := int(src[length-1])
	if length < unPadding {
		return src, ErrUnPadding
	}
	return src[:(length - unPadding)], nil
}

func zerosPadding(src []byte, blockSize int) []byte {
	paddingCount := blockSize - len(src)%blockSize
	if paddingCount == 0 {
		return src
	} else {
		return append(src, bytes.Repeat([]byte{byte(0)}, paddingCount)...)
	}
}

func zerosUnPadding(src []byte) ([]byte, error) {
	for i := len(src) - 1; ; i-- {
		if src[i] != 0 {
			return src[:i+1], nil
		}
	}
}

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

package aes_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/cryptos/aes"
	"github.com/aacfactory/fns/commons/cryptos/ciphers"
	"testing"
	"time"
)

func TestCBC(t *testing.T) {
	cbc, cbcErr := aes.NewCBC([]byte("1234512345123451"), []byte("6789067890678906"), ciphers.PKCS7)
	if cbcErr != nil {
		t.Error(cbcErr)
		return
	}
	p := []byte(time.Now().String())
	e, encodeErr := cbc.Encrypt(p)
	if encodeErr != nil {
		t.Error(encodeErr)
		return
	}
	pp, decodeErr := cbc.Decrypt(e)
	if decodeErr != nil {
		t.Error(decodeErr)
		return
	}
	fmt.Println(string(p) == string(pp), string(p), string(pp))
}

func TestEBC(t *testing.T) {
	ebc, ebcErr := aes.NewEBC([]byte("1234512345123451"), ciphers.PKCS7)
	if ebcErr != nil {
		t.Error(ebcErr)
		return
	}
	p := []byte(time.Now().String())
	e, encodeErr := ebc.Encrypt(p)
	if encodeErr != nil {
		t.Error(encodeErr)
		return
	}
	pp, decodeErr := ebc.Decrypt(e)
	if decodeErr != nil {
		t.Error(decodeErr)
		return
	}
	fmt.Println(string(p) == string(pp), string(p), string(pp))
}

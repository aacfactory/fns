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

package sm2_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/aacfactory/fns/commons/cryptos/sm2"
	"github.com/aacfactory/json"
	"testing"
	"time"
)

func TestExchange(t *testing.T) {
	ida := []byte("A")
	priA, priAErr := sm2.GenerateKey(rand.Reader)
	if priAErr != nil {
		t.Error(priAErr)
		return
	}
	priATemp, _ := sm2.GenerateKey(rand.Reader)
	priB, priBErr := sm2.GenerateKey(rand.Reader)
	if priBErr != nil {
		t.Error(priBErr)
		return
	}
	priBTemp, _ := sm2.GenerateKey(rand.Reader)
	idb := []byte("B")

	k1, k1s1, k1s2, e1 := sm2.KeyExchangeResponder(64, ida, idb, priB, &priA.PublicKey, priBTemp, &priATemp.PublicKey)
	if e1 != nil {
		t.Error(e1)
		return
	}
	k2, k2s1, k2s2, e2 := sm2.KeyExchangeViaInitiator(64, ida, idb, priA, &priB.PublicKey, priATemp, &priBTemp.PublicKey)
	if e2 != nil {
		t.Error(e2)
		return
	}
	fmt.Println(bytes.Equal(k1, k2), base64.StdEncoding.EncodeToString(k1))
	fmt.Println(bytes.Equal(k1s1, k2s1), bytes.Equal(k1s2, k2s2))
	fmt.Println(len(k1s1))
	fmt.Println(base64.StdEncoding.EncodeToString(k1s1))
	p, encodeErr := json.Marshal(map[string][]byte{"r": k1s1, "v": k1s2})
	fmt.Println(string(p), encodeErr)
	p, encodeErr = json.Marshal(time.Now())
	fmt.Println(string(p), encodeErr)
}

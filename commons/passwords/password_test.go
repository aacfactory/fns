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

package passwords_test

import (
	"fmt"
	"github.com/aacfactory/fns/commons/passwords"
	"github.com/aacfactory/json"
	"testing"
)

func TestPassword(t *testing.T) {
	plain := passwords.Password("pass")
	hashed, hashErr := plain.BcryptHash()
	fmt.Println(hashed, hashErr, hashed.Compare("pass"), hashed.Compare("abc"))
	p, encodeErr := json.Marshal(plain)
	fmt.Println(string(p), encodeErr)
	decodeErr := json.Unmarshal(p, &plain)
	fmt.Println(plain, decodeErr)
}

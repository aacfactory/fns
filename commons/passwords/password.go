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

package passwords

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"golang.org/x/crypto/bcrypt"
)

type Password string

func (pass Password) BcryptHash() (bp BcryptPassword, err error) {
	b, bErr := bcrypt.GenerateFromPassword(bytex.FromString(pass.String()), bcrypt.DefaultCost)
	if bErr != nil {
		err = errors.Warning("password: generate bcrypt hash failed").WithCause(bErr).WithMeta("plain", pass.String())
		return
	}
	bp = BcryptPassword(b)
	return
}

func (pass Password) String() string {
	return string(pass)
}

type BcryptPassword string

func (pass BcryptPassword) Compare(plain string) (ok bool) {
	ok = bcrypt.CompareHashAndPassword(bytex.FromString(pass.String()), bytex.FromString(plain)) == nil
	return
}

func (pass BcryptPassword) String() string {
	return string(pass)
}

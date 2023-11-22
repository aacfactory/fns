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

package authorizations

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"strconv"
)

type Id []byte

func (id Id) Int() int64 {
	if len(id) == 0 {
		return 0
	}
	v, err := strconv.ParseInt(id.String(), 10, 64)
	if err != nil {
		panic(errors.Warning("authorizations: get int value from id failed").WithCause(err).WithMeta("id", id.String()))
		return 0
	}
	return v
}

func (id Id) String() string {
	return bytex.ToString(id)
}

func (id Id) Exist() (ok bool) {
	ok = len(id) > 0
	return
}

func StringId(id []byte) Id {
	return id
}

func IntId(id int64) Id {
	return bytex.FromString(strconv.FormatInt(id, 10))
}

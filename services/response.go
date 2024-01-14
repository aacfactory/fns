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

package services

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/objects"
)

type Response interface {
	objects.Object
}

func NewResponse(src any) Response {
	if src == nil {
		return objects.New(nil)
	}
	if _, isEmpty := src.(Empty); isEmpty {
		return objects.New(nil)
	}
	return objects.New(src)
}

// ValueOfResponse
// type of T must be struct value or slice, can not be ptr
func ValueOfResponse[T any](response Response) (v T, err error) {
	v, err = objects.Value[T](response)
	if err != nil {
		err = errors.Warning("fns: get value of response failed").WithCause(err)
		return
	}
	return
}

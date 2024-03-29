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

package context

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/objects"
)

func UserValue[T any](ctx Context, key []byte) (v T, has bool, err error) {
	vv := ctx.UserValue(key)
	if vv == nil {
		return
	}
	v, has = vv.(T)
	if has {
		return
	}
	v, err = objects.Value[T](objects.New(vv))
	if err != nil {
		err = errors.Warning("fns: get context user value failed").WithCause(err).WithMeta("key", string(key))
		return
	}
	has = true
	return
}

func LocalValue[T any](ctx Context, key []byte) (v T, has bool) {
	vv := ctx.LocalValue(key)
	if vv == nil {
		return
	}
	v, has = vv.(T)
	return
}

func Value[T any](ctx Context, key any) (v T, has bool) {
	vv := ctx.Value(key)
	if vv == nil {
		return
	}
	v, has = vv.(T)
	return
}

type valueContext struct {
	Context
	key any
	val any
}

func (c *valueContext) Value(key any) any {
	if c.key == key {
		return c.val
	}
	return c.Context.Value(key)
}

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

package fast

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/context"
	"github.com/valyala/fasthttp"
)

type Context struct {
	*fasthttp.RequestCtx
	locals context.Entries
}

func (ctx *Context) UserValue(key []byte) any {
	return ctx.RequestCtx.UserValueBytes(key)
}

func (ctx *Context) ScanUserValue(key []byte, val any) (has bool, err error) {
	v := ctx.RequestCtx.UserValue(key)
	if v == nil {
		return
	}
	s := scanner.New(v)
	err = s.Scan(val)
	if err != nil {
		err = errors.Warning("fns: scan context value failed").WithMeta("key", bytex.ToString(key)).WithCause(err)
		return
	}
	has = true
	return
}

func (ctx *Context) SetUserValue(key []byte, val any) {
	ctx.RequestCtx.SetUserValueBytes(key, val)
}

func (ctx *Context) UserValues(fn func(key []byte, val any)) {
	ctx.RequestCtx.VisitUserValues(fn)
}

func (ctx *Context) LocalValue(key []byte) any {
	v := ctx.locals.Get(key)
	if v != nil {
		return v
	}
	return nil
}

func (ctx *Context) ScanLocalValue(key []byte, val any) (has bool, err error) {
	v := ctx.LocalValue(key)
	if v == nil {
		return
	}
	s := scanner.New(v)
	err = s.Scan(val)
	if err != nil {
		err = errors.Warning("fns: scan context value failed").WithMeta("key", bytex.ToString(key)).WithCause(err)
		return
	}
	has = true
	return
}

func (ctx *Context) SetLocalValue(key []byte, val any) {
	ctx.locals.Set(key, val)
}

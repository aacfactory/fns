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
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/scanner"
	"sync"
	"unsafe"
)

var (
	pool = sync.Pool{}
)

func Acquire(ctx context.Context) Context {
	cached := pool.Get()
	if cached == nil {
		return &context_{
			Context: ctx,
			entries: make(Entries, 0, 1),
			locals:  make(Entries, 0, 1),
		}
	}
	v := cached.(*context_)
	v.Context = ctx
	return v
}

func Release(ctx context.Context) {
	v, ok := ctx.(*context_)
	if ok {
		v.Context = nil
		v.entries.Reset()
		v.locals.Reset()
		pool.Put(v)
	}
}

type Context interface {
	context.Context
	UserValue(key []byte) any
	ScanUserValue(key []byte, val any) (has bool, err error)
	SetUserValue(key []byte, val any)
	UserValues(fn func(key []byte, val any))
	LocalValue(key []byte) any
	ScanLocalValue(key []byte, val any) (has bool, err error)
	SetLocalValue(key []byte, val any)
}

type context_ struct {
	context.Context
	entries Entries
	locals  Entries
}

func (c *context_) UserValue(key []byte) any {
	v := c.entries.Get(key)
	if v != nil {
		return v
	}
	parent, ok := c.Context.(Context)
	if ok {
		return parent.UserValue(key)
	}
	return nil
}

func (c *context_) ScanUserValue(key []byte, val any) (has bool, err error) {
	v := c.UserValue(key)
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

func (c *context_) SetUserValue(key []byte, val any) {
	c.entries.Set(key, val)
}

func (c *context_) UserValues(fn func(key []byte, val any)) {
	parent, ok := c.Context.(Context)
	if ok {
		parent.UserValues(fn)
	}
	c.entries.Foreach(fn)
}

func (c *context_) LocalValue(key []byte) any {
	v := c.locals.Get(key)
	if v != nil {
		return v
	}
	parent, ok := c.Context.(Context)
	if ok {
		return parent.LocalValue(key)
	}
	return nil
}

func (c *context_) ScanLocalValue(key []byte, val any) (has bool, err error) {
	v := c.LocalValue(key)
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

func (c *context_) SetLocalValue(key []byte, val any) {
	c.locals.Set(key, val)
}

func (c *context_) Value(key any) any {
	switch k := key.(type) {
	case []byte:
		v := c.entries.Get(k)
		if v == nil {
			v = c.locals.Get(k)
			if v == nil {
				return c.Context.Value(key)
			}
		}
		return v
	case string:
		s := unsafe.Slice(unsafe.StringData(k), len(k))
		v := c.entries.Get(s)
		if v == nil {
			v = c.locals.Get(s)
			if v == nil {
				return c.Context.Value(key)
			}
		}
		return v
	default:
		break
	}
	return c.Context.Value(key)
}

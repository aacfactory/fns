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

package fns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Context interface {
	context.Context
	Namespace() (ns string)
	FnName() (name string)
	RequestId() (id string)
	Authorization() (value string)
	PutUser(user User)
	User() (user User, has bool)
	Meta() (meta ContextMeta)
	Log() (log Logs)
	Discovery() (discovery Discovery)
	Timeout() (has bool)
	// Warp
	// 在已执行的FN中调用另一个FN时进行Context转换
	Warp(namespace string, fnName string) (ctx Context)
}

type ContextMeta interface {
	json.Marshaler
	json.Unmarshaler
	Exists(key string) (has bool)
	Put(key string, value interface{})
	Get(key string, value interface{}) (err error)
	GetString(key string) (value string, has bool)
	GetInt(key string) (value int, has bool)
	GetInt32(key string) (value int32, has bool)
	GetInt64(key string) (value int64, has bool)
	GetFloat32(key string) (value float32, has bool)
	GetFloat64(key string) (value float64, has bool)
	GetBool(key string) (value bool, has bool)
	GetTime(key string) (value time.Time, has bool)
	GetDuration(key string) (value time.Duration, has bool)
	Encode() (value []byte)
	Decode(value []byte) (ok bool)
}

// +-------------------------------------------------------------------------------------------------------------------+

func newContext(ctx context.Context, log Logs, discovery Discovery, namespace string, fnName string, authorization string, requestId string) Context {
	return &fnContext{
		Context:       ctx,
		namespace:     namespace,
		fnName:        fnName,
		requestId:     requestId,
		authorization: authorization,
		user:          nil,
		log:           log,
		meta:          newFnsContextMeta(),
		discovery:     discovery,
	}
}

type fnContext struct {
	context.Context
	namespace     string
	fnName        string
	requestId     string
	authorization string
	user          User
	meta          ContextMeta
	log           Logs
	discovery     Discovery
}

func (ctx *fnContext) Namespace() (ns string) {
	ns = ctx.namespace
	return
}

func (ctx *fnContext) FnName() (name string) {
	name = ctx.fnName
	return
}

func (ctx *fnContext) RequestId() (id string) {
	id = ctx.requestId
	return
}

func (ctx *fnContext) Authorization() (value string) {
	value = ctx.authorization
	return
}

func (ctx *fnContext) PutUser(user User) {
	if user == nil {
		return
	}
	ctx.user = user
	return
}

func (ctx *fnContext) User() (user User, has bool) {
	if ctx.user == nil {
		return
	}
	user = ctx.user
	has = true
	return
}

func (ctx *fnContext) Log() (log Logs) {
	log = ctx.log
	return
}

func (ctx *fnContext) Meta() (meta ContextMeta) {
	meta = ctx.meta
	return
}

func (ctx *fnContext) Discovery() (discovery Discovery) {
	discovery = ctx.discovery
	return
}

func (ctx *fnContext) Timeout() (has bool) {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return
	}
	has = deadline.Before(time.Now())
	return
}

func (ctx *fnContext) Warp(namespace string, fnName string) (forked Context) {
	if ctx.requestId == "" {
		panic("context can not warp, cause no fn request id")
		return
	}
	log := LogWith(ctx.log, LogF("ns", namespace), LogF("fn", fnName), LogF("rid", ctx.RequestId()))
	forked = &fnContext{
		Context:       ctx.Context,
		namespace:     namespace,
		fnName:        fnName,
		requestId:     ctx.requestId,
		authorization: ctx.authorization,
		user:          ctx.user,
		log:           log,
		meta:          newFnsContextMetaFromMeta(ctx.meta),
		discovery:     ctx.discovery,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

func newFnsContextMetaFromMeta(meta ContextMeta) (nm *fnsContextMeta) {
	data, _ := meta.MarshalJSON()
	nm = &fnsContextMeta{
		obj: NewJsonObjectFromBytes(data),
	}
	return
}

func newFnsContextMeta() ContextMeta {
	return &fnsContextMeta{
		obj: NewJsonObject(),
	}
}

type fnsContextMeta struct {
	obj *JsonObject
}

func (meta *fnsContextMeta) Exists(key string) (has bool) {
	has = meta.obj.Contains(key)
	return
}

func (meta *fnsContextMeta) Put(key string, value interface{}) {
	if key == "" || value == nil {
		return
	}
	_ = meta.obj.Put(key, value)
}

func (meta *fnsContextMeta) Get(key string, value interface{}) (err error) {
	if !meta.Exists(key) {
		err = fmt.Errorf("%s was not found", key)
		return
	}
	getErr := meta.obj.Get(key, value)
	if getErr != nil {
		err = fmt.Errorf("get %s failed", key)
	}
	return
}

func (meta *fnsContextMeta) GetString(key string) (value string, has bool) {
	if !meta.Exists(key) {
		return
	}
	getErr := meta.obj.Get(key, &value)
	if getErr != nil {
		return
	}
	has = true
	return
}

func (meta *fnsContextMeta) GetInt(key string) (value int, has bool) {
	if !meta.Exists(key) {
		return
	}
	getErr := meta.obj.Get(key, &value)
	if getErr != nil {
		return
	}
	has = true
	return
}

func (meta *fnsContextMeta) GetInt32(key string) (value int32, has bool) {
	if !meta.Exists(key) {
		return
	}
	getErr := meta.obj.Get(key, &value)
	if getErr != nil {
		return
	}
	has = true
	return
}

func (meta *fnsContextMeta) GetInt64(key string) (value int64, has bool) {
	if !meta.Exists(key) {
		return
	}
	getErr := meta.obj.Get(key, &value)
	if getErr != nil {
		return
	}
	has = true
	return
}

func (meta *fnsContextMeta) GetFloat32(key string) (value float32, has bool) {
	if !meta.Exists(key) {
		return
	}
	getErr := meta.obj.Get(key, &value)
	if getErr != nil {
		return
	}
	has = true
	return
}

func (meta *fnsContextMeta) GetFloat64(key string) (value float64, has bool) {
	if !meta.Exists(key) {
		return
	}
	getErr := meta.obj.Get(key, &value)
	if getErr != nil {
		return
	}
	has = true
	return
}

func (meta *fnsContextMeta) GetBool(key string) (value bool, has bool) {
	if !meta.Exists(key) {
		return
	}
	getErr := meta.obj.Get(key, &value)
	if getErr != nil {
		return
	}
	has = true
	return
}

func (meta *fnsContextMeta) GetTime(key string) (value time.Time, has bool) {
	if !meta.Exists(key) {
		return
	}
	getErr := meta.obj.Get(key, &value)
	if getErr != nil {
		return
	}
	has = true
	return
}

func (meta *fnsContextMeta) GetDuration(key string) (value time.Duration, has bool) {
	if !meta.Exists(key) {
		return
	}
	getErr := meta.obj.Get(key, &value)
	if getErr != nil {
		return
	}
	has = true
	return
}

func (meta *fnsContextMeta) MarshalJSON() (b []byte, err error) {
	b = meta.obj.raw
	return
}

func (meta *fnsContextMeta) UnmarshalJSON(b []byte) (err error) {
	meta.obj = NewJsonObjectFromBytes(b)
	return
}

func (meta *fnsContextMeta) Encode() (value []byte) {
	value = Sign(meta.obj.raw)
	return
}

func (meta *fnsContextMeta) Decode(value []byte) (ok bool) {
	if !Verify(value) {
		return
	}
	idx := bytes.LastIndexByte(value, '.')
	src := value[:idx]
	meta.obj = NewJsonObjectFromBytes(src)
	ok = true
	return
}

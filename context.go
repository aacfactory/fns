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
	"context"
	"fmt"
	"github.com/tidwall/gjson"
	"strings"
	"time"
)

type Context interface {
	context.Context
	Meta() (meta ContextMeta)
	Log() (log Logs)
	Bus() (bus FnBus)
	Timeout() (has bool)
	WithFnRequest(namespace string, fnName string, requestId string) (ctx Context)
	WithFnFork(namespace string, fnName string) (ctx Context)
}

type ContextMeta interface {
	Exists(key string) (has bool)
	Put(key string, value interface{})
	Get(key string) (value interface{}, has bool)
	GetString(key string) (value string, has bool)
	GetInt(key string) (value int, has bool)
	GetInt32(key string) (value int32, has bool)
	GetInt64(key string) (value int64, has bool)
	GetFloat32(key string) (value float32, has bool)
	GetFloat64(key string) (value float64, has bool)
	GetBool(key string) (value bool, has bool)
	GetBytes(key string) (value []byte, has bool)
	GetTime(key string) (value time.Time, has bool)
	GetDuration(key string) (value time.Duration, has bool)
	Namespace() (ns string)
	FnName() (name string)
	RequestId() (id string)
	PutAuthorization(value string)
	Authorization() (value string)
	PutUser(user User)
	User() (user User, has bool)
	Values() (values map[string]interface{})
}

// +-------------------------------------------------------------------------------------------------------------------+

func newFnsContext(ctx context.Context, log Logs, bus FnBus) Context {
	return &fnsContext{
		Context: ctx,
		parent:  nil,
		meta:    newFnsContextMeta(),
		log:     log,
		bus:     bus,
	}
}

type fnsContext struct {
	context.Context
	parent *fnsContext
	log    Logs
	meta   ContextMeta
	bus    FnBus
}

func (ctx *fnsContext) Log() (log Logs) {
	log = ctx.log
	return
}

func (ctx *fnsContext) Meta() (meta ContextMeta) {
	meta = ctx.meta
	return
}

func (ctx *fnsContext) Bus() (bus FnBus) {
	bus = ctx.bus
	return
}

func (ctx *fnsContext) Timeout() (has bool) {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return
	}
	has = deadline.Before(time.Now())
	return
}

func (ctx *fnsContext) WithFnRequest(namespace string, fnName string, requestId string) (fnContext Context) {
	if ctx.meta.Exists(contextMetaFnRequestIdKey) {
		panic("context can not with fn request again")
		return
	}
	meta := newFnsContextMetaFromMeta(ctx.meta)
	meta.Put(contextMetaNamespaceKey, namespace)
	meta.Put(contextMetaFnNameKey, fnName)
	meta.Put(contextMetaFnRequestIdKey, requestId)
	log := LogWith(ctx.Log(), LogF("ns", namespace), LogF("fn", fnName), LogF("rid", requestId))
	fnContext = &fnsContext{
		Context: ctx.Context,
		parent:  ctx,
		meta:    meta,
		log:     log,
		bus:     ctx.bus,
	}
	return
}

func (ctx *fnsContext) WithFnFork(namespace string, fnName string) (forked Context) {
	if !ctx.meta.Exists(contextMetaFnRequestIdKey) || ctx.parent == nil {
		panic("context can not with fn fork, cause no fn request id")
		return
	}
	meta := newFnsContextMetaFromMeta(ctx.meta)
	meta.Put(contextMetaNamespaceKey, namespace)
	meta.Put(contextMetaFnNameKey, fnName)
	log := LogWith(ctx.parent.Log(), LogF("ns", namespace), LogF("fn", fnName), LogF("rid", meta.RequestId()))
	forked = &fnsContext{
		Context: ctx.Context,
		parent:  ctx,
		meta:    meta,
		log:     log,
		bus:     ctx.bus,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

const (
	contextMetaNamespaceKey     = "__ns"
	contextMetaFnNameKey        = "__fn"
	contextMetaFnRequestIdKey   = "__rid"
	contextMetaUserKey          = "__user"
	contextMetaAuthorizationKey = "__auth"
)

func newFnsContextMetaFromMeta(meta ContextMeta) (nm *fnsContextMeta) {
	values := meta.Values()
	copied := make(map[string]interface{})
	for k, v := range values {
		copied[k] = v
	}
	nm = &fnsContextMeta{
		values: copied,
	}
	return
}

func newFnsContextMeta() ContextMeta {
	return &fnsContextMeta{
		values: make(map[string]interface{}),
	}
}

type fnsContextMeta struct {
	values map[string]interface{}
}

func (meta *fnsContextMeta) Exists(key string) (has bool) {
	_, has = meta.values[key]
	return
}

func (meta *fnsContextMeta) Put(key string, value interface{}) {
	meta.values[key] = value
}

func (meta *fnsContextMeta) Get(key string) (value interface{}, has bool) {
	value, has = meta.values[key]
	return
}

func (meta *fnsContextMeta) GetString(key string) (value string, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.(string)
	return
}

func (meta *fnsContextMeta) GetInt(key string) (value int, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.(int)
	return
}

func (meta *fnsContextMeta) GetInt32(key string) (value int32, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.(int32)
	return
}

func (meta *fnsContextMeta) GetInt64(key string) (value int64, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.(int64)
	return
}

func (meta *fnsContextMeta) GetFloat32(key string) (value float32, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.(float32)
	return
}

func (meta *fnsContextMeta) GetFloat64(key string) (value float64, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.(float64)
	return
}

func (meta *fnsContextMeta) GetBool(key string) (value bool, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.(bool)
	return
}

func (meta *fnsContextMeta) GetBytes(key string) (value []byte, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.([]byte)
	return
}

func (meta *fnsContextMeta) GetTime(key string) (value time.Time, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.(time.Time)
	return
}

func (meta *fnsContextMeta) GetDuration(key string) (value time.Duration, has bool) {
	value0, has0 := meta.values[key]
	if !has0 {
		return
	}
	value, has = value0.(time.Duration)
	return
}

func (meta *fnsContextMeta) Namespace() (ns string) {
	ns, _ = meta.GetString(contextMetaNamespaceKey)
	return
}

func (meta *fnsContextMeta) FnName() (name string) {
	name, _ = meta.GetString(contextMetaFnNameKey)
	return
}

func (meta *fnsContextMeta) RequestId() (id string) {
	id, _ = meta.GetString(contextMetaFnRequestIdKey)
	return
}

func (meta *fnsContextMeta) PutAuthorization(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	meta.Put(contextMetaAuthorizationKey, value)
}

func (meta *fnsContextMeta) Authorization() (value string) {
	value, _ = meta.GetString(contextMetaAuthorizationKey)
	return
}

func (meta *fnsContextMeta) PutUser(user User) {
	if user == nil {
		return
	}
	meta.Put(contextMetaUserKey, user)
	return
}

func (meta *fnsContextMeta) User() (user User, has bool) {
	value0, has0 := meta.values[contextMetaUserKey]
	if !has0 {
		return
	}
	user, has = value0.(User)
	return
}

func (meta *fnsContextMeta) Values() (values map[string]interface{}) {
	values = meta.values
	return
}

func newContextMetaFromJson(data []byte) (meta ContextMeta) {
	meta = newFnsContextMeta()
	result := gjson.ParseBytes(data)
	if !result.Exists() {
		return
	}
	resultMap := result.Map()
	for key, value := range resultMap {
		if key == contextMetaNamespaceKey {
			meta.Put(contextMetaNamespaceKey, value.String())
			continue
		}
		if key == contextMetaFnNameKey {
			meta.Put(contextMetaFnNameKey, value.String())
			continue
		}
		if key == contextMetaFnRequestIdKey {
			meta.Put(contextMetaFnRequestIdKey, value.String())
			continue
		}
		if key == contextMetaUserKey {
			user := NewUser()
			userErr := user.UnmarshalJSON([]byte(value.Raw))
			if userErr != nil {
				panic(fmt.Errorf("decode context user failed, %s, %v", string(value.Raw), userErr))
			}
			meta.Put(contextMetaUserKey, user)
			continue
		}
		if key == contextMetaAuthorizationKey {
			meta.Put(contextMetaAuthorizationKey, value.String())
			continue
		}
		switch value.Type {
		case gjson.String:
			meta.Put(key, value.String())
		case gjson.Number:
			if strings.Index(value.Raw, ".") > 0 {
				meta.Put(key, value.Float())
			} else {
				meta.Put(key, value.Int())
			}
		case gjson.False:
			meta.Put(key, false)
		case gjson.True:
			meta.Put(key, true)
		default:

		}
	}
	return meta
}

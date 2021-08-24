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
	sc "context"
	"fmt"
	"github.com/aacfactory/fns/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"time"
)

func WithNamespace(ctx Context, namespace string) Context {
	ctx0 := ctx.(*context)
	return &context{
		Context:       ctx0,
		id:            ctx0.RequestId(),
		authorization: ctx0.Authorization(),
		user:          ctx0.User(),
		meta:          ctx0.meta.fork(),
		log:           ctx0.Log().With("namespace", namespace),
		discovery:     ctx0.discovery,
	}
}

func WithFn(ctx Context, fnName string) Context {
	ctx0 := ctx.(*context)
	return &context{
		Context:       ctx0,
		id:            ctx0.RequestId(),
		authorization: ctx0.Authorization(),
		user:          ctx0.User(),
		meta:          ctx0.meta.fork(),
		log:           ctx0.Log().With("fn", fnName),
		discovery:     ctx0.discovery,
	}
}

type context struct {
	sc.Context
	id            string
	authorization string
	user          User
	meta          *contextMeta
	log           logs.Logger
	discovery     ServiceDiscovery
}

func (ctx *context) RequestId() (id string) {
	id = ctx.id
	return
}

func (ctx *context) Authorization() (value string) {
	value = ctx.authorization
	return
}

func (ctx *context) User() (user User) {
	user = ctx.user
	return
}

func (ctx *context) Meta() (meta ContextMeta) {
	meta = ctx.meta
	return
}

func (ctx *context) Log() (log logs.Logger) {
	log = ctx.log
	return
}

func (ctx *context) ServiceProxy(namespace string) (proxy ServiceProxy, err error) {
	proxy, err = ctx.discovery.Proxy(namespace)
	return
}

func (ctx *context) Timeout() (has bool) {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return
	}
	has = deadline.Before(time.Now())
	return
}

type contextMeta struct {
	obj *json.Object
}

func (meta *contextMeta) fork() *contextMeta {
	return &contextMeta{
		obj: json.NewObjectFromBytes(meta.obj.Raw()),
	}
}

func (meta *contextMeta) Exists(key string) (has bool) {
	has = meta.obj.Contains(key)
	return
}

func (meta *contextMeta) Put(key string, value interface{}) {
	if key == "" || value == nil {
		return
	}
	_ = meta.obj.Put(key, value)
}

func (meta *contextMeta) Get(key string, value interface{}) (err error) {
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

func (meta *contextMeta) GetString(key string) (value string, has bool) {
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

func (meta *contextMeta) GetInt(key string) (value int, has bool) {
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

func (meta *contextMeta) GetInt32(key string) (value int32, has bool) {
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

func (meta *contextMeta) GetInt64(key string) (value int64, has bool) {
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

func (meta *contextMeta) GetFloat32(key string) (value float32, has bool) {
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

func (meta *contextMeta) GetFloat64(key string) (value float64, has bool) {
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

func (meta *contextMeta) GetBool(key string) (value bool, has bool) {
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

func (meta *contextMeta) GetTime(key string) (value time.Time, has bool) {
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

func (meta *contextMeta) GetDuration(key string) (value time.Duration, has bool) {
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

func (meta *contextMeta) MarshalJSON() (b []byte, err error) {
	b = meta.obj.Raw()
	return
}

func (meta *contextMeta) UnmarshalJSON(b []byte) (err error) {
	err = meta.obj.UnmarshalJSON(b)
	return
}

func (meta *contextMeta) Encode() (value []byte) {
	value = secret.Sign(meta.obj.Raw(), secretKey)
	return
}

func (meta *contextMeta) Decode(value []byte) (ok bool) {
	if !secret.Verify(value, secretKey) {
		return
	}
	idx := bytes.LastIndexByte(value, '.')
	src := value[:idx]
	err := meta.obj.UnmarshalJSON(src)
	if err != nil {
		return
	}
	ok = true
	return
}

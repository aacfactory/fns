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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/fns/secret"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
	"time"
)

func WithNamespace(ctx Context, namespace string) Context {
	ctx0 := ctx.(*context)
	if ctx0.namespace == "" {
		ctx0.namespace = namespace
		ctx0.log = ctx0.Log().With("namespace", namespace)
		return ctx0
	}
	return &context{
		Context:       ctx0,
		namespace:     namespace,
		id:            ctx0.RequestId(),
		authorization: ctx0.Authorization(),
		user:          ctx0.User(),
		meta:          ctx0.meta.fork(),
		log:           ctx0.Log().With("namespace", namespace).With("fn", ""),
		discovery:     ctx0.discovery,
	}
}

func WithFn(ctx Context, fnName string) Context {
	ctx0 := ctx.(*context)
	ctx0.log = ctx0.Log().With("fn", fnName)
	return ctx0
}

func withDiscovery(ctx Context, discovery ServiceDiscovery) Context {
	ctx.(*context).discovery = discovery
	return ctx
}

func newContext(ctx sc.Context, id string) *context {
	return &context{
		Context: ctx,
		id:      id,
		user:    newUser(),
		meta:    newContextMeta(),
	}
}

type context struct {
	sc.Context
	namespace     string
	id            string
	authorization []byte
	user          User
	meta          *contextMeta
	log           logs.Logger
	validate      *validator.Validate
	discovery     ServiceDiscovery
}

func (ctx *context) RequestId() (id string) {
	id = ctx.id
	return
}

func (ctx *context) Authorization() (value []byte) {
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
	proxy, err = ctx.discovery.Proxy(ctx, namespace)
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

func (ctx *context) Validate(v interface{}) (err errors.CodeError) {
	if ctx.validate == nil {
		err = errors.NotImplemented("context Validate: not implemented")
		return
	}
	validateErr := ctx.validate.Struct(v)
	if validateErr == nil {
		return
	}
	validationErrors, ok := validateErr.(validator.ValidationErrors)
	if !ok {
		err = errors.ServiceError(fmt.Sprintf("context Validate: %v", validateErr))
		return
	}
	err = errors.BadRequest("argument is invalid")
	for _, validationError := range validationErrors {
		sf := validationError.Namespace()
		exp := sf[strings.Index(sf, ".")+1:]
		key, message := commons.ValidateFieldMessage(reflect.TypeOf(v), exp)
		if key == "" {
			err = errors.ServiceError(fmt.Sprintf("context Validate: json tag of %s was not founed", sf))
			return
		}
		if message == "" {
			err = errors.ServiceError(fmt.Sprintf("context Validate: message tag of %s was not founed", sf))
			return
		}
		err = err.WithMeta(key, message)
	}
	return
}

func newContextMeta() *contextMeta {
	return &contextMeta{
		obj: json.NewObject(),
	}
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

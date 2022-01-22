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
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
	"time"
)

type appRuntime struct {
	clusterMode    bool
	publicAddress  string
	appLog         logs.Logger
	nsLog          logs.Logger
	fnLog          logs.Logger
	validate       *validator.Validate
	discovery      ServiceDiscovery
	authorizations Authorizations
	permissions    Permissions
	httpClients    *HttpClients
	serviceMeta    ServiceMeta
}

func (app *appRuntime) ClusterMode() (ok bool) {
	ok = app.clusterMode
	return
}

func (app *appRuntime) PublicAddress() (address string) {
	address = app.publicAddress
	return
}

func (app *appRuntime) Log() (log logs.Logger) {
	if app.fnLog != nil {
		log = app.fnLog
	} else if app.nsLog != nil {
		log = app.nsLog
	} else {
		log = app.appLog
	}
	return
}

func (app *appRuntime) Validate(v interface{}) (err errors.CodeError) {
	if app.validate == nil {
		err = errors.NotImplemented("fns Validation: validate not implemented")
		return
	}
	validateErr := app.validate.Struct(v)
	if validateErr == nil {
		return
	}
	validationErrors, ok := validateErr.(validator.ValidationErrors)
	if !ok {
		err = errors.New(555, "***WARNING***", fmt.Sprintf("fns Validation: validate failed")).WithCause(validateErr)
		return
	}
	err = errors.BadRequest("fns Validation: argument is invalid")
	for _, validationError := range validationErrors {
		sf := validationError.Namespace()
		exp := sf[strings.Index(sf, ".")+1:]
		key, message := commons.ValidateFieldMessage(reflect.TypeOf(v), exp)
		if key == "" {
			err = errors.New(555, "***WARNING***", fmt.Sprintf("fns Validation: validate failed, json tag of %s was not founed", sf))
			return
		}
		if message == "" {
			err = errors.New(555, "***WARNING***", fmt.Sprintf("fns Validation: validate failed, message tag of %s was not founed", sf))
			return
		}
		err = err.WithMeta(key, message)
	}
	return
}

func (app *appRuntime) ServiceProxy(ctx Context, namespace string) (proxy ServiceProxy, err error) {
	proxy, err = app.discovery.Proxy(ctx, namespace)
	return
}

func (app *appRuntime) ServiceMeta() (meta ServiceMeta) {
	meta = app.serviceMeta
	return
}

func (app *appRuntime) Authorizations() (authorizations Authorizations) {
	authorizations = app.authorizations
	return
}

func (app *appRuntime) Permissions() (permissions Permissions) {
	permissions = app.permissions
	return
}

func (app *appRuntime) HttpClient() (client HttpClient) {
	client = &httpClient{
		client: app.httpClients.next(),
	}
	return
}

func WithServiceMeta(ctx Context, meta ServiceMeta) Context {
	ctx0 := ctx.(*context)
	ctx0.app.serviceMeta = meta
	return ctx
}

func WithNamespace(ctx Context, namespace string) Context {
	ctx0 := ctx.(*context)
	if ctx0.app.nsLog == nil {
		ctx0.app.nsLog = ctx0.app.appLog.With("namespace", namespace)
		return ctx0
	}
	app := &appRuntime{
		clusterMode:    ctx0.app.clusterMode,
		publicAddress:  ctx0.app.publicAddress,
		appLog:         ctx0.app.appLog,
		nsLog:          ctx0.app.appLog.With("namespace", namespace),
		fnLog:          nil,
		validate:       ctx0.app.validate,
		discovery:      ctx0.app.discovery,
		authorizations: ctx0.app.authorizations,
		permissions:    ctx0.app.permissions,
		httpClients:    ctx0.app.httpClients,
	}
	return &context{
		Context: ctx0.Context,
		id:      ctx0.RequestId(),
		user:    ctx0.User(),
		meta:    ctx0.meta.fork(),
		app:     app,
	}
}

func WithFn(ctx Context, fn string) Context {
	ctx0 := ctx.(*context)
	if ctx0.app.nsLog == nil {
		panic("can not call fns.WithFn before call fns.WithNamespace")
		return ctx0
	}
	ctx0.app.fnLog = ctx0.app.nsLog.With("fn", fn)
	return ctx0
}

func WithInternalRequest(ctx Context) Context {
	ctx0 := ctx.(*context)
	ctx0.internal = true
	return ctx0
}

func newContext(_ctx sc.Context, internal bool, id string, authorization []byte, metaData []byte, app *appRuntime) (ctx *context, err error) {
	meta, metaErr := newContextMeta(metaData)
	if metaErr != nil {
		err = metaErr
		return
	}
	ctx = &context{
		Context:  _ctx,
		id:       id,
		internal: internal,
		user:     newUser(authorization),
		meta:     meta,
		app:      app,
	}
	return
}

type context struct {
	sc.Context
	id       string
	internal bool
	user     User
	meta     *contextMeta
	app      *appRuntime
}

func (ctx *context) RequestId() (id string) {
	id = ctx.id
	return
}

func (ctx *context) InternalRequested() (ok bool) {
	ok = ctx.internal
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

func (ctx *context) App() (app AppRuntime) {
	app = ctx.app
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

func newContextMeta(p []byte) (meta *contextMeta, err error) {
	obj := json.NewObject()
	err = obj.UnmarshalJSON(p)
	if err != nil {
		return
	}
	meta = &contextMeta{
		obj: obj,
	}
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

func (meta *contextMeta) Remove(key string) {
	_ = meta.obj.Remove(key)
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

func (meta *contextMeta) SetExactProxyServiceAddress(namespace string, address string) {
	meta.Put(fmt.Sprintf("%s_%s", serviceExactProxyMetaKeyPrefix, namespace), address)
}

func (meta *contextMeta) GetExactProxyServiceAddress(namespace string) (address string, has bool) {
	address, has = meta.GetString(fmt.Sprintf("%s_%s", serviceExactProxyMetaKeyPrefix, namespace))
	return
}

func (meta *contextMeta) DelExactProxyServiceAddress(namespace string) {
	meta.Remove(fmt.Sprintf("%s_%s", serviceExactProxyMetaKeyPrefix, namespace))
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
	value = meta.obj.Raw()
	return
}

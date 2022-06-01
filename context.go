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
	"github.com/aacfactory/fns/internal/commons"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"time"
)

// +-------------------------------------------------------------------------------------------------------------------+

type ContextData interface {
	json.Marshaler
	json.Unmarshaler
	Contains(key string) (has bool)
	Put(key string, value interface{})
	Get(key string, value interface{}) (has bool)
	Remove(key string)
	GetString(key string) (value string, has bool)
	GetInt(key string) (value int, has bool)
	GetInt64(key string) (value int64, has bool)
	GetFloat32(key string) (value float32, has bool)
	GetFloat64(key string) (value float64, has bool)
	GetBool(key string) (value bool, has bool)
	GetTime(key string) (value time.Time, has bool)
	GetDuration(key string) (value time.Duration, has bool)
	Copy() (o ContextData)
	Merge(o ContextData)
}

func newContextData(values *json.Object) ContextData {
	return &contextData{
		values: values,
	}
}

type contextData struct {
	values *json.Object
}

func (data *contextData) Contains(key string) (has bool) {
	has = data.values.Contains(key)
	return
}

func (data *contextData) Put(key string, value interface{}) {
	if key == "" || value == nil {
		return
	}
	err := data.values.Put(key, value)
	if err != nil {
		panic(fmt.Errorf("%+v", errors.Warning(fmt.Sprintf("fns: can not put %v into context data", key)).WithCause(err)))
	}
}

func (data *contextData) Get(key string, value interface{}) (has bool) {
	has = data.Contains(key)
	if !has {
		return
	}
	err := data.values.Get(key, value)
	if err != nil {
		panic(fmt.Errorf("%+v", errors.Warning(fmt.Sprintf("fns: can not get %v from context data", key)).WithCause(err)))
	}
	return
}

func (data *contextData) Remove(key string) {
	_ = data.values.Remove(key)
}

func (data *contextData) GetString(key string) (value string, has bool) {
	has = data.Get(key, &value)
	return
}

func (data *contextData) GetInt(key string) (value int, has bool) {
	has = data.Get(key, &value)
	return
}

func (data *contextData) GetInt64(key string) (value int64, has bool) {
	has = data.Get(key, &value)
	return
}

func (data *contextData) GetFloat32(key string) (value float32, has bool) {
	has = data.Get(key, &value)
	return
}

func (data *contextData) GetFloat64(key string) (value float64, has bool) {
	has = data.Get(key, &value)
	return
}

func (data *contextData) GetBool(key string) (value bool, has bool) {
	has = data.Get(key, &value)
	return
}

func (data *contextData) GetTime(key string) (value time.Time, has bool) {
	has = data.Get(key, &value)
	return
}

func (data *contextData) GetDuration(key string) (value time.Duration, has bool) {
	has = data.Get(key, &value)
	return
}

func (data *contextData) Copy() (o ContextData) {
	o = newContextData(json.NewObjectFromBytes(data.values.Raw()))
	return
}

func (data *contextData) Merge(o ContextData) {
	if o == nil {
		return
	}
	oValues := o.(*contextData).values
	if oValues == nil {
		return
	}
	err := data.values.Merge(oValues)
	if err != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: can not merge from another context data").WithCause(err)))
	}
}

func (data *contextData) MarshalJSON() (p []byte, err error) {
	p, err = data.values.MarshalJSON()
	return
}

func (data *contextData) UnmarshalJSON(p []byte) (err error) {
	err = data.values.UnmarshalJSON(p)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Context interface {
	sc.Context
	Request() (request Request)
	CanAccessInternal() (ok bool)
	Data() (data ContextData)
	Log() (log logs.Logger)
	Tracer() (tracer Tracer)
	CurrentServiceComponent(key string, component interface{}) (err errors.CodeError)
	Runtime() (rt Runtime)
	Fork(ctx sc.Context) (o Context)
}

func newContext(ctx sc.Context, request Request, data ContextData, rt Runtime) (v *context) {
	v = &context{
		Context:           ctx,
		request:           request,
		data:              data,
		log:               rt.Log(),
		tracer:            newTracer(request.Id()),
		serviceComponents: make(map[string]ServiceComponent),
		runtime:           rt,
	}
	return
}

type context struct {
	sc.Context
	request           Request
	data              ContextData
	log               logs.Logger
	tracer            Tracer
	serviceComponents map[string]ServiceComponent
	runtime           Runtime
}

func (ctx *context) Request() (request Request) {
	request = ctx.request
	return
}

func (ctx *context) CanAccessInternal() (ok bool) {
	if ctx.request.Internal() {
		ok = true
		return
	}
	if ctx.tracer.SpanSize() > 0 {
		ok = true
	}
	return
}

func (ctx *context) Data() (data ContextData) {
	data = ctx.data
	return
}

func (ctx *context) Log() (log logs.Logger) {
	log = ctx.log
	return
}

func (ctx *context) Tracer() (tracer Tracer) {
	tracer = ctx.tracer
	return
}

func (ctx *context) CurrentServiceComponent(key string, component interface{}) (err errors.CodeError) {
	if ctx.serviceComponents == nil || len(ctx.serviceComponents) == 0 {
		err = errors.Warning(fmt.Sprintf("fns: there is no any component in context"))
		return
	}
	comp, has := ctx.serviceComponents[key]
	if !has {
		err = errors.Warning(fmt.Sprintf("fns: there is no %s component in context", key))
		return
	}
	cpErr := commons.CopyInterface(component, comp)
	if cpErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: found %s component but load failed", key)).WithCause(cpErr)
		return
	}
	return
}

func (ctx *context) Runtime() (rt Runtime) {
	rt = ctx.runtime
	return
}

func (ctx *context) Fork(c sc.Context) (o Context) {
	o = &context{
		Context:           c,
		request:           ctx.request,
		data:              ctx.data.Copy(),
		log:               ctx.log.With("forked", true),
		tracer:            newTracer(ctx.request.Id()),
		serviceComponents: ctx.serviceComponents,
		runtime:           ctx.runtime,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

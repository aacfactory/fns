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
	"github.com/aacfactory/cluster"
	"github.com/aacfactory/eventbus"
	"github.com/aacfactory/logs"
	"time"
)

type Context interface {
	context.Context
	Log() (log Logs)
	Meta() (meta ContextMeta)
	Eventbus() (bus eventbus.Eventbus)
	Shared() (shared ContextShared)
}

type ContextShared interface {
	Map(name string) (sm cluster.SharedMap)
	Counter(name string) (counter cluster.SharedCounter, err error)
	Locker(name string, timeout time.Duration) (locker cluster.SharedLocker)
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
}

type FnContext interface {
	Context
	RequestId() (id string)
	FnAddress() (addr string)
	User() (user User, has bool)
	SetUser(user User)
}

// +-------------------------------------------------------------------------------------------------------------------+

func newFnsContext(ctx context.Context, log Logs, bus eventbus.Eventbus, c cluster.Cluster) Context {
	return &fnsContext{
		Context: ctx,
		log:     log,
		meta:    newFnsContextMeta(),
		bus:     bus,
		cluster: c,
	}
}

type fnsContext struct {
	context.Context
	log     Logs
	meta    ContextMeta
	bus     eventbus.Eventbus
	cluster cluster.Cluster
}

func (ctx *fnsContext) Log() (log Logs) {
	log = ctx.log
	return
}

func (ctx *fnsContext) Meta() (meta ContextMeta) {
	meta = ctx.meta
	return
}

func (ctx *fnsContext) Eventbus() (bus eventbus.Eventbus) {
	bus = ctx.bus
	return
}

func (ctx *fnsContext) Shared() (shared ContextShared) {
	if ctx.cluster == nil {
		panic(fmt.Errorf("fns is not in cluster mode"))
	}
	shared = &fnsContextShared{
		cluster: ctx.cluster,
	}
	return
}

type fnsContextShared struct {
	cluster cluster.Cluster
}

func (shared *fnsContextShared) Map(name string) (sm cluster.SharedMap) {
	sm = shared.cluster.GetMap(name)
	return
}

func (shared *fnsContextShared) Counter(name string) (counter cluster.SharedCounter, err error) {
	counter, err = shared.cluster.Counter(name)
	return
}

func (shared *fnsContextShared) Locker(name string, timeout time.Duration) (locker cluster.SharedLocker) {
	locker = shared.cluster.GetLock(name, timeout)
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

func newFnsFnContext(fnAddr string, requestId string, ctx Context, clusterMode bool) FnContext {
	var shared ContextShared = nil
	if clusterMode {
		shared = ctx.Shared()
	}
	subLog := logs.With(ctx.Log(), logs.F("fn", fnAddr), logs.F("rid", requestId))
	return &fnsFnContext{
		Context:   context.TODO(),
		log:       subLog,
		meta:      newFnsContextMeta(),
		bus:       ctx.Eventbus(),
		shared:    shared,
		requestId: requestId,
		fnAddress: fnAddr,
		user:      nil,
	}
}

type fnsFnContext struct {
	context.Context
	log       Logs
	meta      ContextMeta
	bus       eventbus.Eventbus
	shared    ContextShared
	requestId string
	fnAddress string
	user      User
}

func (ctx *fnsFnContext) Log() (log Logs) {
	log = ctx.log
	return
}

func (ctx *fnsFnContext) Meta() (meta ContextMeta) {
	meta = ctx.meta
	return
}

func (ctx *fnsFnContext) Eventbus() (bus eventbus.Eventbus) {
	bus = ctx.bus
	return
}

func (ctx *fnsFnContext) Shared() (shared ContextShared) {
	if ctx.shared == nil {
		panic(fmt.Errorf("fns is not in cluster mode"))
	}
	shared = ctx.shared
	return
}

func (ctx *fnsFnContext) RequestId() (id string) {
	id = ctx.requestId
	return
}

func (ctx *fnsFnContext) FnAddress() (addr string) {
	addr = ctx.fnAddress
	return
}

func (ctx *fnsFnContext) setUser(user User) {
	ctx.user = user
}

func (ctx *fnsFnContext) User() (user User, has bool) {
	if ctx.user != nil {
		user = ctx.user
		has = true
		return
	}
	return
}

func (ctx *fnsFnContext) SetUser(user User) {
	ctx.user = user
	return
}

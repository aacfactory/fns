package fns

import (
	"context"
	"fmt"
	"time"
)

type Context interface {
	context.Context
	Log() (log Logs)
	Meta() (meta ContextMeta)
	Eventbus() (bus Eventbus)
	Shared() (shared ContextShared, err error)
}

type ContextShared interface {
	Map(name string) (sm SharedMap)
	Counter(name string) (counter SharedCounter, err error)
	Locker(name string, timeout time.Duration) (locker SharedLocker)
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
	AuthCredentials() (authCredentials AuthCredentials, has bool)
	User() (user User, has bool)
}

// +-------------------------------------------------------------------------------------------------------------------+

func newFnsContext(ctx context.Context, log Logs, bus Eventbus, cluster Cluster) Context {
	return &fnsContext{
		Context: ctx,
		log:     log,
		meta:    newFnsContextMeta(),
		bus:     bus,
		cluster: cluster,
	}
}

type fnsContext struct {
	context.Context
	log     Logs
	meta    ContextMeta
	bus     Eventbus
	cluster Cluster
}

func (ctx *fnsContext) Log() (log Logs) {
	log = ctx.log
	return
}

func (ctx *fnsContext) Meta() (meta ContextMeta) {
	meta = ctx.meta
	return
}

func (ctx *fnsContext) Eventbus() (bus Eventbus) {
	bus = ctx.bus
	return
}

func (ctx *fnsContext) Shared() (shared ContextShared, err error) {
	if ctx.cluster == nil {
		err = fmt.Errorf("fns is not in cluster mode")
		return
	}
	shared = &fnsContextShared{
		cluster: ctx.cluster,
	}
	return
}

type fnsContextShared struct {
	cluster Cluster
}

func (shared *fnsContextShared) Map(name string) (sm SharedMap) {
	sm = shared.cluster.GetMap(name)
	return
}

func (shared *fnsContextShared) Counter(name string) (counter SharedCounter, err error) {
	counter, err = shared.cluster.Counter(name)
	return
}

func (shared *fnsContextShared) Locker(name string, timeout time.Duration) (locker SharedLocker) {
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

func newFnsFnContext(requestId string, ctx context.Context, log Logs, bus Eventbus, cluster Cluster) FnContext {
	return &fnsFnContext{
		Context:         newFnsContext(ctx, log, bus, cluster),
		requestId:       requestId,
		authCredentials: nil,
		user:            nil,
	}
}

type fnsFnContext struct {
	Context
	requestId       string
	authCredentials AuthCredentials
	user            User
}

func (ctx *fnsFnContext) RequestId() (id string) {
	id = ctx.requestId
	return
}

func (ctx *fnsFnContext) setAuthCredentials(authCredentials AuthCredentials) {
	ctx.authCredentials = authCredentials
}

func (ctx *fnsFnContext) AuthCredentials() (authCredentials AuthCredentials, has bool) {
	if ctx.authCredentials != nil {
		authCredentials = ctx.authCredentials
		has = true
		return
	}
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

package fns

import (
	"context"
	"github.com/aacfactory/cluster"
	"github.com/aacfactory/eventbus"
	"github.com/aacfactory/logs"
	"time"
)

type Context interface {
	context.Context
	Log() (log logs.Logs)
	Meta() (meta ContextMeta)
	Eventbus() (bus eventbus.Eventbus)
	Shared() (shared ContextShared, err error)
}

type ContextShared interface {
	Map(name string) (sm cluster.SharedMap)
	Counter(name string) (counter cluster.SharedCounter)
	Locker(name string) (locker cluster.SharedLocker)
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
	User() (user interface{}, has bool)
}

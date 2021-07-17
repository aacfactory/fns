package fns

import "context"

type DeliveryOptions interface {
	Add(key string, value string)
	Put(key string, value []string)
	Get(key string) (string, bool)
	Keys() []string
	Empty() bool
	Values(key string) ([]string, bool)
	Remove(key string)
	AddTag(tags ...string)
}

type Eventbus interface {
	Send(address string, v interface{}, options ...DeliveryOptions) (err error)
	Request(address string, v interface{}, options ...DeliveryOptions) (reply ReplyFuture)
	RegisterHandler(address string, handler EventHandler, tags ...string) (err error)
	RegisterLocalHandler(address string, handler EventHandler, tags ...string) (err error)
	Start(context context.Context)
	Close(context context.Context)
}

type EventHead interface {
	Add(key string, value string)
	Put(key string, value []string)
	Get(key string) (string, bool)
	Keys() []string
	Empty() bool
	Values(key string) ([]string, bool)
	Remove(key string)
}

type Event interface {
	Head() EventHead
	Body() []byte
}

type EventHandler func(event Event) (result interface{}, err error)

type ReplyFuture interface {
	Get(v interface{}) (err error)
}

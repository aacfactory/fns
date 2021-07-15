package fns

import (
	"crypto/tls"
	"time"
)

type Status interface {
	Ok() bool
	Closing() bool
}

type Registration interface {
	NodeId() (nodeId string)
	NodeName() (nodeName string)
	Id() (id string)
	Group() (group string)
	Name() (name string)
	Status() (status Status)
	Protocol() (protocol string)
	Address() (address string)
	Tags() (tags []string)
	Meta() (meta Meta)
	TLS() (registrationTLS RegistrationTLS)
}

type Meta interface {
	Put(key string, value string)
	Get(key string) (value string, has bool)
	Rem(key string)
	Keys() (keys []string)
	Empty() (ok bool)
	Merge(o ...Meta)
}

type RegistrationTLS interface {
	Enable() bool
	VerifySSL() bool
	CA() string
	ServerCert() string
	ServerKey() string
	ClientCert() string
	ClientKey() string
	ToServerTLSConfig() (config *tls.Config, err error)
	ToClientTLSConfig() (config *tls.Config, err error)
}

type RegistrationEventType interface {
	IsPut() bool
	IsRem() bool
}

type RegistrationEvent interface {
	Registration() (registration Registration)
	Type() (eventType RegistrationEventType)
}

type ServiceDiscovery interface {
	Publish(group string, name string, protocol string, address string, tags []string, meta Meta, registrationTLS RegistrationTLS) (registration Registration, err error)
	UnPublish(registration Registration) (err error)
	Get(group string, name string, tags ...string) (registration Registration, has bool, err error)
	GetALL(group string, name string, tags ...string) (registrations []Registration, has bool, err error)
}

type Cluster interface {
	ServiceDiscovery
	Join() (err error)
	Leave() (err error)
	GetMap(name string) (m SharedMap)
	GetLock(name string, timeout time.Duration) (locker SharedLocker)
	Counter(name string) (counter SharedCounter, err error)
	Close()
}

type SharedMap interface {
	Put(key string, value interface{}, ttl time.Duration) (err error)
	Get(key string, value interface{}) (has bool, err error)
	Rem(key string) (err error)
	Keys() (keys []string, err error)
	Clean() (err error)
}

type SharedCounter interface {
	Get() (v int64, err error)
	IncrementAndGet() (v int64, err error)
	DecrementAndGet() (v int64, err error)
	AddAndGet(d int64) (v int64, err error)
	CompareAndSet(expected int64, v int64) (ok bool, err error)
	Close()
}

type SharedLocker interface {
	Lock() (err error)
	UnLock() (err error)
}


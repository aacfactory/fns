package fns

import (
	"context"
	"github.com/aacfactory/cluster"
	"github.com/aacfactory/eventbus"
	"github.com/aacfactory/logs"
)

type Context interface {
	context.Context
	Log() (log logs.Logs)
	Eventbus() (bus eventbus.Eventbus)
	SharedMap(name string) (sm cluster.SharedMap)
	SharedCounter(name string) (counter cluster.SharedCounter)
	SharedLocker(name string) (locker cluster.SharedLocker)
	Discovery() (discovery cluster.ServiceDiscovery)
}

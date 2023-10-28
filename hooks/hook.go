package hooks

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
)

type Options struct {
	Log    logs.Logger
	Config configures.Config
}

type Hook interface {
	workers.NamedTask
	Construct(options Options) (err error)
}

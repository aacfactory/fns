package hooks

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/workers"
)

type Config map[string]json.RawMessage

func (config Config) Get(name string) (v configures.Config, err error) {
	if len(config) == 0 {
		v, _ = configures.NewJsonConfig([]byte{'{', '}'})
		return
	}
	p, has := config[name]
	if !has {
		v, _ = configures.NewJsonConfig([]byte{'{', '}'})
		return
	}
	v, err = configures.NewJsonConfig(p)
	if err != nil {
		err = errors.Warning("fns: get hook config failed").WithCause(err).WithMeta("hook", name)
		return
	}
	return
}

type Options struct {
	Log    logs.Logger
	Config configures.Config
}

type Hook interface {
	workers.NamedTask
	Construct(options Options) (err error)
}

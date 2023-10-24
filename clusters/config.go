package clusters

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
)

type Config map[string]json.RawMessage

func (config Config) Get(name string) (v configures.Config, err error) {
	p, exist := config[name]
	if !exist || len(p) == 0 {
		v, _ = configures.NewJsonConfig([]byte{'{', '}'})
		return
	}
	v, err = configures.NewJsonConfig(p)
	if err != nil {
		err = errors.Warning("fns: get cluster config failed").WithMeta("name", name).WithCause(err)
		return
	}
	return
}

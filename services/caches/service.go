package caches

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/services"
)

var (
	endpointName = []byte("caches")
	getFnName    = []byte("get")
	setFnName    = []byte("set")
	remFnName    = []byte("remove")
)

func NewWithStore(store Store) services.Service {
	return &service{
		Abstract: services.NewAbstract(string(endpointName), true, store),
		sn:       store.Name(),
	}
}

// New
// use @cache, and param must implement KeyParam.
// @cache get
// @cache set 10
// @cache remove
// @cache get-set 10
func New() services.Service {
	return NewWithStore(&defaultStore{})
}

type service struct {
	services.Abstract
	sn string
}

func (s *service) Construct(options services.Options) (err error) {
	err = s.Abstract.Construct(options)
	if err != nil {
		return
	}
	c, has := s.Components().Get(s.sn)
	if !has {
		err = errors.Warning("fns: caches service construct failed").WithCause(fmt.Errorf("store was not found"))
		return
	}
	store, ok := c.(Store)
	if !ok {
		err = errors.Warning("fns: caches service construct failed").WithCause(fmt.Errorf("%s is not store", s.sn))
		return
	}
	s.AddFunction(&getFn{
		store: store,
	})
	s.AddFunction(&setFn{
		store: store,
	})
	s.AddFunction(&removeFn{
		store: store,
	})
	return
}

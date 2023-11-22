/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

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
// use @cache, and param must implement KeyParam. unit of ttl is seconds and default value is 10 seconds.
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

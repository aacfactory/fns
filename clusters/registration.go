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

package clusters

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services"
	"sync"
)

type Registration struct {
	values sync.Map
}

func (r *Registration) Add(endpoint *Endpoint) {
	key := endpoint.Name()
	var eps *Endpoints
	exist, has := r.values.Load(key)
	if has {
		eps = exist.(*Endpoints)
	} else {
		eps = &Endpoints{
			values: nil,
			length: 0,
			lock:   sync.RWMutex{},
		}
		r.values.Store(key, eps)
	}
	eps.Add(endpoint)
}

func (r *Registration) Remove(name string, id string) {
	exist, has := r.values.Load(name)
	if !has {
		return
	}
	eps := exist.(*Endpoints)
	eps.Remove(bytex.FromString(id))
}

func (r *Registration) Get(name []byte, id []byte) *Endpoint {
	key := bytex.ToString(name)
	exist, has := r.values.Load(key)
	if !has {
		return nil
	}
	eps := exist.(*Endpoints)
	return eps.Get(id)
}

func (r *Registration) Range(name []byte, interval versions.Interval) *Endpoint {
	key := bytex.ToString(name)
	exist, has := r.values.Load(key)
	if !has {
		return nil
	}
	eps := exist.(*Endpoints)
	return eps.Range(interval)
}

func (r *Registration) MaxOne(name []byte) *Endpoint {
	key := bytex.ToString(name)
	exist, has := r.values.Load(key)
	if !has {
		return nil
	}
	eps := exist.(*Endpoints)
	return eps.MaxOne()
}

func (r *Registration) Infos() (v services.EndpointInfos) {
	r.values.Range(func(key, value any) bool {
		eps := value.(*Endpoints)
		vv := eps.Infos()
		if len(vv) > 0 {
			v = append(v, vv...)
		}
		return true
	})
	return
}

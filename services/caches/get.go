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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/json"
)

func Get(ctx context.Context, param interface{}) (p []byte, has bool, err error) {
	if param == nil {
		err = errors.Warning("fns: get cache failed").WithCause(fmt.Errorf("param is nil"))
		return
	}
	kp, ok := param.(KeyParam)
	if !ok {
		err = errors.Warning("fns: get cache failed").WithCause(fmt.Errorf("param dose not implement caches.KeyParam"))
		return
	}
	key, keyErr := kp.CacheKey(ctx)
	if keyErr != nil {
		err = errors.Warning("fns: get cache failed").WithCause(keyErr)
		return
	}
	eps := runtime.Endpoints(ctx)
	response, doErr := eps.Request(ctx, endpointName, getFnName, getFnParam{
		Key: bytex.ToString(key),
	}, services.WithInternalRequest())
	if doErr != nil {
		err = doErr
		return
	}
	result, resultErr := services.ValueOfResponse[getResult](response)
	if resultErr != nil {
		err = errors.Warning("fns: get cache failed").WithCause(resultErr)
		return
	}
	p = result.Value
	has = result.Has
	return
}

type getFnParam struct {
	Key string `json:"key"`
}

type getResult struct {
	Has   bool            `json:"has"`
	Value json.RawMessage `json:"value"`
}

type getFn struct {
	store Store
}

func (fn *getFn) Name() string {
	return string(getFnName)
}

func (fn *getFn) Internal() bool {
	return true
}

func (fn *getFn) Readonly() bool {
	return false
}

func (fn *getFn) Handle(r services.Request) (v interface{}, err error) {
	if !r.Param().Exist() {
		err = errors.Warning("fns: get cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	param, paramErr := services.ValueOfParam[getFnParam](r.Param())
	if paramErr != nil {
		err = errors.Warning("fns: get cache failed").WithCause(paramErr)
		return
	}
	key := bytex.FromString(param.Key)
	if len(key) == 0 {
		err = errors.Warning("fns: get cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	value, has, getErr := fn.store.Get(r, key)
	if getErr != nil {
		err = errors.Warning("fns: get cache failed").WithCause(getErr)
		return
	}
	v = getResult{
		Has:   has,
		Value: value,
	}
	return
}

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
)

func Remove(ctx context.Context, param interface{}) (err error) {
	if param == nil {
		err = errors.Warning("fns: remove cache failed").WithCause(fmt.Errorf("param is nil"))
		return
	}
	kp, ok := param.(KeyParam)
	if !ok {
		err = errors.Warning("fns: remove cache failed").WithCause(fmt.Errorf("param dose not implement caches.KeyParam"))
		return
	}
	key, keyErr := kp.CacheKey(ctx)
	if keyErr != nil {
		err = errors.Warning("fns: remove cache failed").WithCause(keyErr)
		return
	}
	eps := runtime.Endpoints(ctx)
	_, doErr := eps.Request(ctx, endpointName, remFnName, removeFnParam{
		Key: bytex.ToString(key),
	}, services.WithInternalRequest())
	if doErr != nil {
		err = doErr
		return
	}
	return
}

type removeFnParam struct {
	Key string `json:"key" avro:"key"`
}

type removeFn struct {
	store Store
}

func (fn *removeFn) Name() string {
	return string(remFnName)
}

func (fn *removeFn) Internal() bool {
	return true
}

func (fn *removeFn) Readonly() bool {
	return false
}

func (fn *removeFn) Handle(r services.Request) (v interface{}, err error) {
	if !r.Param().Valid() {
		err = errors.Warning("fns: remove cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	param, paramErr := services.ValueOfParam[removeFnParam](r.Param())
	if paramErr != nil {
		err = errors.Warning("fns: remove cache failed").WithCause(paramErr)
		return
	}
	key := bytex.FromString(param.Key)
	if len(key) == 0 {
		err = errors.Warning("fns: remove cache failed").WithCause(errors.Warning("param is invalid"))
		return
	}
	removeErr := fn.store.Remove(r, key)
	if removeErr != nil {
		err = errors.Warning("fns: remove cache failed").WithCause(removeErr)
		return
	}
	return
}

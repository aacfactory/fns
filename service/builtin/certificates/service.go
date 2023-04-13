/*
 * Copyright 2021 Wang Min Xiang
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
 */

package certificates

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
)

const (
	name     = "certificates"
	createFn = "create"
	getFn    = "get"
	removeFn = "remove"
)

func Service(store Store) service.Service {
	return &_service{
		Abstract: service.NewAbstract(name, true, convertStoreToComponent(store)),
		store:    store,
	}
}

type _service struct {
	service.Abstract
	store Store
}

func (svc _service) Handle(ctx context.Context, fn string, argument service.Argument) (v interface{}, err errors.CodeError) {
	switch fn {
	case createFn:
		param := Certificate{}
		paramErr := argument.As(&param)
		if paramErr != nil {
			err = errors.Warning("certificates: create failed").WithCause(paramErr)
			return
		}
		err = svc.store.Create(ctx, &param)
		break
	case getFn:
		param := ""
		paramErr := argument.As(&param)
		if paramErr != nil {
			err = errors.Warning("certificates: get failed").WithCause(paramErr)
			return
		}
		v, err = svc.store.Get(ctx, param)
		break
	case removeFn:
		param := ""
		paramErr := argument.As(&param)
		if paramErr != nil {
			err = errors.Warning("certificates: remove failed").WithCause(paramErr)
			return
		}
		err = svc.store.Remove(ctx, param)
		break
	default:
		err = errors.NotFound("certificates: fn was not found")
		break
	}
	return
}

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

package permissions

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
)

const (
	name      = "permissions"
	enforceFn = "enforce"
)

func Service(enforcer Enforcer) (v service.Service) {
	if enforcer == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("permissions: service requires enforcer component")))
		return
	}
	v = &service_{
		Abstract: service.NewAbstract(name, true, enforcer),
	}
	return
}

type service_ struct {
	service.Abstract
	enforcer Enforcer
}

func (svc *service_) Build(options service.Options) (err error) {
	err = svc.Abstract.Build(options)
	if err != nil {
		return
	}
	if svc.Components() == nil || len(svc.Components()) != 1 {
		err = errors.Warning("permissions: build failed").WithCause(errors.Warning("permissions: enforcer is required"))
		return
	}
	for _, component := range svc.Components() {
		enforcer, ok := component.(Enforcer)
		if !ok {
			err = errors.Warning("permissions: build failed").WithCause(errors.Warning("permissions: enforcer is required"))
			return
		}
		svc.enforcer = enforcer
	}
	return
}

func (svc *service_) Handle(ctx context.Context, fn string, argument service.Argument) (v interface{}, err errors.CodeError) {
	switch fn {
	case enforceFn:
		param := EnforceParam{}
		paramErr := argument.As(&param)
		if paramErr != nil {
			err = errors.Warning("permissions: enforce failed").WithCause(paramErr)
			break
		}
		v, err = svc.enforcer.Enforce(ctx, param)
		break
	default:
		err = errors.Warning("permissions: fn was not found")
		break
	}
	return
}

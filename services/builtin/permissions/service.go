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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/services"
)

const (
	name      = "permissions"
	enforceFn = "enforce"
)

func Service(enforcer Enforcer) (v services.Service) {
	if enforcer == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("permissions: service requires enforcer component")))
		return
	}
	v = &service{
		Abstract: services.NewAbstract(name, true, enforcer),
	}
	return
}

type service struct {
	services.Abstract
	enforcer Enforcer
}

func (svc *service) Construct(options services.Options) (err error) {
	err = svc.Abstract.Construct(options)
	if err != nil {
		return
	}
	if svc.Components() == nil || len(svc.Components()) != 1 {
		err = errors.Warning("permissions: construct failed").WithMeta("endpoint", svc.Name()).WithCause(errors.Warning("permissions: enforcer is required"))
		return
	}
	for _, component := range svc.Components() {
		enforcer, ok := component.(Enforcer)
		if !ok {
			err = errors.Warning("permissions: construct failed").WithMeta("endpoint", svc.Name()).WithCause(errors.Warning("permissions: enforcer is required"))
			return
		}
		svc.enforcer = enforcer
	}
	return
}

func (svc *service) Handle(ctx services.Request) (v interface{}, err error) {
	_, fn := ctx.Fn()
	switch bytex.ToString(fn) {
	case enforceFn:
		param := EnforceParam{}
		paramErr := ctx.Argument().As(&param)
		if paramErr != nil {
			err = errors.Warning("permissions: enforce failed").WithMeta("endpoint", svc.Name()).WithMeta("fn", string(fn)).WithCause(paramErr)
			break
		}
		v, err = svc.enforcer.Enforce(ctx, param)
		break
	default:
		err = errors.NotFound("permissions: fn was not found").WithMeta("endpoint", svc.Name()).WithMeta("fn", string(fn))
		break
	}
	return
}

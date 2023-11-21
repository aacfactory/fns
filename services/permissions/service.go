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
	"github.com/aacfactory/fns/services"
)

const (
	endpointName  = "permissions"
	enforceFnName = "enforce"
)

type enforceFn struct {
	enforcer Enforcer
}

func (fn *enforceFn) Name() string {
	return enforceFnName
}

func (fn *enforceFn) Internal() bool {
	return true
}

func (fn *enforceFn) Readonly() bool {
	return false
}

func (fn *enforceFn) Handle(r services.Request) (v interface{}, err error) {
	param := EnforceParam{}
	paramErr := r.Param().Scan(&param)
	if paramErr != nil {
		err = errors.BadRequest("permissions: invalid enforce param").WithCause(paramErr)
		return
	}
	v, err = fn.enforcer.Enforce(r, param)
	if err != nil {
		err = errors.ServiceError("permissions: enforce failed").WithCause(err)
		return
	}
	return
}

func Service(enforcer Enforcer) (v services.Service) {
	if enforcer == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("permissions: service requires enforcer component")))
		return
	}
	v = &service{
		Abstract: services.NewAbstract(endpointName, true, enforcer),
	}
	return
}

// service
// use @permission
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
	var enforcer Enforcer
	has := false
	for _, component := range svc.Components() {
		enforcer, has = component.(Enforcer)
		if has {
			break
		}
	}
	if enforcer == nil {
		err = errors.Warning("permissions: service need token encoder component")
		return
	}
	svc.Abstract.AddFunction(&enforceFn{
		enforcer: enforcer,
	})
	return
}

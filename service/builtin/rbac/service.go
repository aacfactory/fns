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

package rbac

import (
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/logs"
)

const (
	Name       = "rbac"
	RoleFn     = "role"
	RolesFn    = "roles"
	ChildrenFn = "children"
	SaveFn     = "save"
	RemoveFn   = "remove"
	BindFn     = "bind"
	UnbindFn   = "unbind"
	BoundsFn   = "bounds"
	EnforceFn  = "enforce"
)

func Service(components ...service.Component) (v service.Service) {
	var store service.Component
	for _, component := range components {
		if component.Name() == "store" {
			store = component
			continue
		}
		if store != nil {
			break
		}
	}
	if store == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: create rbac service failed").WithCause(fmt.Errorf("store is nil"))))
	}
	v = &_service_{
		components: map[string]service.Component{
			"store": store,
		},
	}
	return
}

type _service_ struct {
	log        logs.Logger
	components map[string]service.Component
}

func (svc *_service_) Name() (name string) {
	name = Name
	return
}

func (svc *_service_) Internal() (internal bool) {
	internal = true
	return
}

func (svc *_service_) Build(options service.Options) (err error) {
	svc.log = options.Log
	if svc.components != nil {
		for cn, component := range svc.components {
			if component == nil {
				continue
			}
			componentCfg, hasConfig := options.Config.Node(cn)
			if !hasConfig {
				componentCfg, _ = configures.NewJsonConfig([]byte("{}"))
			}
			err = component.Build(service.ComponentOptions{
				Log:    options.Log.With("component", cn),
				Config: componentCfg,
			})
			if err != nil {
				err = errors.Warning("fns: build rbac service failed").WithCause(err)
				return
			}
		}
	}
	return
}

func (svc *_service_) Components() (components map[string]service.Component) {
	components = svc.components
	return
}

func (svc *_service_) Document() (doc service.Document) {
	return
}

func (svc *_service_) Handle(ctx context.Context, fn string, argument service.Argument) (v interface{}, err errors.CodeError) {
	switch fn {
	case RoleFn:
		fnArgument := RoleArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("rbac: invalid request argument").WithCause(argumentErr).WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		v, err = role(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		break
	case RolesFn:
		fnArgument := RolesArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("rbac: invalid request argument").WithCause(argumentErr).WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		v, err = roles(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		break
	case ChildrenFn:
		fnArgument := ChildrenArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("rbac: invalid request argument").WithCause(argumentErr).WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		v, err = children(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		break
	case SaveFn:
		fnArgument := SaveArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("rbac: invalid request argument").WithCause(argumentErr).WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		err = save(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		break
	case RemoveFn:
		fnArgument := RemoveArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("rbac: invalid request argument").WithCause(argumentErr).WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		err = remove(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		break
	case BoundsFn:
		fnArgument := BoundsArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("rbac: invalid request argument").WithCause(argumentErr).WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		v, err = bounds(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		break
	case BindFn:
		fnArgument := BindArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("rbac: invalid request argument").WithCause(argumentErr).WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		err = bind(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		break
	case UnbindFn:
		fnArgument := UnbindArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("rbac: invalid request argument").WithCause(argumentErr).WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		err = unbind(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		break
	case EnforceFn:
		fnArgument := EnforceArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("rbac: invalid request argument").WithCause(argumentErr).WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		v, err = enforce(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "rbac").WithMeta("fn", fn)
			return
		}
		break
	default:
		err = errors.NotFound("rbac: fn was not found").WithMeta("service", "rbac").WithMeta("fn", fn)
		break
	}
	return
}

func (svc *_service_) Close() {
	if svc.components != nil && len(svc.components) > 0 {
		for _, component := range svc.components {
			component.Close()
		}
	}
	if svc.log.DebugEnabled() {
		svc.log.Debug().Message("rbac: closed")
	}
}

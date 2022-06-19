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
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/logs"
)

var (
	store Store = nil
)

func RegisterStore(s Store) {
	if s == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: register permissions components failed").WithCause(fmt.Errorf("store is nil"))))
	}
	store = s
}

func Service() (v service.Service) {
	if store == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: create permissions service failed").WithCause(fmt.Errorf("store is nil"))))
	}
	v = &_service_{
		components: map[string]service.Component{
			"store": &storeComponent{
				store: store,
			},
		},
	}
	return
}

type _service_ struct {
	log        logs.Logger
	components map[string]service.Component
}

func (svc *_service_) Name() (name string) {
	name = "permissions"
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
				componentCfg, _ = configuares.NewJsonConfig([]byte("{}"))
			}
			err = component.Build(service.ComponentOptions{
				Log:    options.Log.With("component", cn),
				Config: componentCfg,
			})
			if err != nil {
				err = errors.Warning("fns: build permissions service failed").WithCause(err)
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
	case "verify":
		fnArgument := VerifyArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("permissions: invalid request argument").WithCause(argumentErr).WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		v, err = verify(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		break
	case "get_user_roles":
		fnArgument := GetUserRolesArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("permissions: invalid request argument").WithCause(argumentErr).WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		v, err = userRoles(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		break
	case "user_bind_roles":
		fnArgument := UserBindRolesArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("permissions: invalid request argument").WithCause(argumentErr).WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		err = userBindRoles(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		v = &service.Empty{}
		break
	case "user_unbind_roles":
		fnArgument := UserUnbindRolesArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("permissions: invalid request argument").WithCause(argumentErr).WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		err = userUnbindRoles(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		v = &service.Empty{}
		break
	case "role":
		fnArgument := GetRoleArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("permissions: invalid request argument").WithCause(argumentErr).WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		v, err = getRole(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		break
	case "roles":
		v, err = getRoles(ctx)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		break
	case "save_role":
		fnArgument := Role{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("permissions: invalid request argument").WithCause(argumentErr).WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		err = saveRole(ctx, &fnArgument)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		v = &service.Empty{}
		break
	case "remove_role":
		fnArgument := RemoveRoleArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("permissions: invalid request argument").WithCause(argumentErr).WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		err = removeRole(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		v = &service.Empty{}
		break
	case "check_user_can_read_resource":
		fnArgument := CheckResourcePermissionArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("permissions: invalid request argument").WithCause(argumentErr).WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		v, err = canReadResource(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		break
	case "check_user_can_write_resource":
		fnArgument := CheckResourcePermissionArgument{}
		argumentErr := argument.As(&fnArgument)
		if argumentErr != nil {
			err = errors.BadRequest("permissions: invalid request argument").WithCause(argumentErr).WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		v, err = canWriteResource(ctx, fnArgument)
		if err != nil {
			err = err.WithMeta("service", "permissions").WithMeta("fn", fn)
			return
		}
		break
	default:
		err = errors.NotFound("permissions: fn was not found").WithMeta("service", "permissions").WithMeta("fn", fn)
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
		svc.log.Debug().Message("permissions: closed")
	}
}

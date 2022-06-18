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

package authorizations

import (
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/logs"
	"golang.org/x/net/context"
)

var (
	encoding = createDefaultTokenEncoding()
	store    = createDiscardTokenStore()
)

func RegisterTokenEncoding(tokenEncoding TokenEncoding) {
	if tokenEncoding == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: register authorizations components failed").WithCause(fmt.Errorf("encoding is nil"))))
	}
	encoding = tokenEncoding
}

func RegisterTokenStore(tokenStore TokenStore) {
	if tokenStore == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: register authorizations components failed").WithCause(fmt.Errorf("store is nil"))))
	}
	store = tokenStore
}

func Service() (v service.Service) {
	v = &_service_{
		components: map[string]service.Component{
			"store": &tokenStoreComponent{
				store: store,
			},
			"encoding": &tokenEncodingComponent{
				encoding: encoding,
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
	name = "authorizations"
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
				err = errors.Warning("fns: build authorizations service failed").WithCause(err)
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
	case "encode":
		param := EncodeParam{}
		asErr := argument.As(&param)
		if asErr != nil {
			err = errors.BadRequest("fns: encode argument failed").WithCause(asErr).WithMeta("service", "authorizations").WithMeta("fn", fn)
			break
		}
		v, err = encode(ctx, param)
		if err != nil {
			err = err.WithMeta("service", "authorizations").WithMeta("fn", fn)
			break
		}
		break
	case "decode":
		param := DecodeParam{}
		asErr := argument.As(&param)
		if asErr != nil {
			err = errors.BadRequest("fns: decode argument failed").WithCause(asErr).WithMeta("service", "authorizations").WithMeta("fn", fn)
			break
		}
		v, err = decode(ctx, param)
		if err != nil {
			err = err.WithMeta("service", "authorizations").WithMeta("fn", fn)
			break
		}
		break
	case "revoke":
		param := RevokeParam{}
		asErr := argument.As(&param)
		if asErr != nil {
			err = errors.BadRequest("fns: revoke argument failed").WithCause(asErr).WithMeta("service", "authorizations").WithMeta("fn", fn)
			break
		}
		v, err = revoke(ctx, param)
		if err != nil {
			err = err.WithMeta("service", "authorizations").WithMeta("fn", fn)
			break
		}
		break
	default:
		err = errors.NotFound("fns: fn was not found").WithMeta("service", "authorizations").WithMeta("fn", fn)
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
		svc.log.Debug().Message("authorizations: closed")
	}
}

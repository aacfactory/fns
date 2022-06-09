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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"golang.org/x/net/context"
)

func Service(encoding TokenEncoding, store TokenStore) (v service.Service) {
	if encoding == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: create authorizations service failed").WithCause(fmt.Errorf("encoding is nil"))))
	}
	if store == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: create authorizations service failed").WithCause(fmt.Errorf("store is nil"))))
	}
	v = &authorizationService{
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

type authorizationService struct {
	components map[string]service.Component
}

func (svc *authorizationService) Name() (name string) {
	name = "authorizations"
	return
}

func (svc *authorizationService) Internal() (internal bool) {
	internal = true
	return
}

func (svc *authorizationService) Build(options service.Options) (err error) {

	return
}

func (svc *authorizationService) Components() (components map[string]service.Component) {
	components = svc.components
	return
}

func (svc *authorizationService) Document() (doc service.Document) {
	return
}

func (svc *authorizationService) Handle(context context.Context, fn string, argument service.Argument) (v interface{}, err errors.CodeError) {
	switch fn {
	case "encode":
		param := EncodeParam{}
		asErr := argument.As(&param)
		if asErr != nil {
			err = errors.BadRequest("fns: decode argument failed").WithCause(asErr).WithMeta("service", "authorizations").WithMeta("fn", fn)
			break
		}

		break
	case "decode":

		break
	case "revoke":

		break
	default:
		err = errors.NotFound("fns: fn was not found").WithMeta("service", "authorizations").WithMeta("fn", fn)
		break
	}
	return
}

func (svc *authorizationService) Close() {

}

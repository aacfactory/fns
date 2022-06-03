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
	sc "context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns"
	"github.com/aacfactory/fns/documents"
)

func Service(encoding TokenEncoding, store TokenStore) (v fns.Service) {
	if encoding == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: new authorizations failed").WithCause(fmt.Errorf("fns: encoding is nil"))))
	}
	if store == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: new authorizations failed").WithCause(fmt.Errorf("fns: store is nil"))))
	}
	v = &service{
		components: map[string]fns.ServiceComponent{
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

type service struct {
	components map[string]fns.ServiceComponent
}

func (s *service) Name() (name string) {
	name = "authorizations"
	return
}

func (s *service) Internal() (internal bool) {
	internal = true
	return
}

func (s *service) Build(env fns.Environments) (err error) {
	for _, component := range s.components {
		err = component.Build(env)
		if err != nil {
			err = errors.Warning("fns: build authorizations service failed").WithCause(err)
			return
		}
	}
	return
}

func (s *service) Components() (components map[string]fns.ServiceComponent) {
	components = s.components
	return
}

func (s *service) Document() (doc *documents.Service) {
	return
}

func (s *service) Handle(ctx fns.Context, fn string, argument fns.Argument, writer fns.ResultWriter) {
	switch fn {
	case "encode":
		param := EncodeParam{}
		asErr := argument.As(&param)
		if asErr != nil {
			writer.Failed(errors.BadRequest("fns: invalid encode param").WithMeta("service", "authorizations").WithMeta("fn", "encode"))
			break
		}
		result, err := encode(ctx, param)
		if err == nil {
			writer.Succeed(result)
		} else {
			writer.Failed(err.WithMeta("service", "authorizations").WithMeta("fn", "encode"))
		}
		break
	case "decode":

		break
	case "revoke":

		break
	default:
		writer.Failed(errors.NotFound(fmt.Sprintf("fns: there is no %s fn in authorizations service", fn)).WithMeta("service", "authorizations"))
		break
	}
	return
}

func (s *service) Shutdown(_ sc.Context) (err error) {
	return
}

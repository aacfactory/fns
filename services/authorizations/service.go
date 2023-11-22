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

package authorizations

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/services"
)

type encodeFn struct {
	encoder TokenEncoder
}

func (fn *encodeFn) Name() string {
	return encodeFnName
}

func (fn *encodeFn) Internal() bool {
	return true
}

func (fn *encodeFn) Readonly() bool {
	return false
}

func (fn *encodeFn) Handle(r services.Request) (v interface{}, err error) {
	param := Authorization{}
	paramErr := r.Param().Scan(&param)
	if paramErr != nil {
		err = errors.BadRequest("authorizations: invalid param")
		return
	}
	token, encodeErr := fn.encoder.Encode(r, param)
	if encodeErr != nil {
		err = errors.ServiceError("authorizations: encode authorization failed").WithCause(encodeErr)
		return
	}
	v = token
	return
}

type decodeFn struct {
	encoder TokenEncoder
}

func (fn *decodeFn) Name() string {
	return decodeFnName
}

func (fn *decodeFn) Internal() bool {
	return true
}

func (fn *decodeFn) Readonly() bool {
	return false
}

func (fn *decodeFn) Handle(r services.Request) (v interface{}, err error) {
	param := Token{}
	paramErr := r.Param().Scan(&param)
	if paramErr != nil {
		err = errors.BadRequest("authorizations: invalid param")
		return
	}
	authorization, decodeErr := fn.encoder.Decode(r, param)
	if decodeErr != nil {
		err = errors.ServiceError("authorizations: decode token failed").WithCause(decodeErr)
		return
	}
	v = authorization
	return
}

func ServiceWithEncoder(encoder TokenEncoder) services.Service {
	return &service{
		Abstract: services.NewAbstract(endpointName, true, encoder),
	}
}

func Service() services.Service {
	return ServiceWithEncoder(DefaultTokenEncoder())
}

// service
// use @authorization
type service struct {
	services.Abstract
}

func (svc *service) Construct(options services.Options) (err error) {
	err = svc.Abstract.Construct(options)
	if err != nil {
		return
	}
	var encoder TokenEncoder
	has := false
	components := svc.Abstract.Components()
	for _, component := range components {
		encoder, has = component.(TokenEncoder)
		if has {
			break
		}
	}
	if encoder == nil {
		err = errors.Warning("authorizations: service need token encoder component")
		return
	}
	svc.AddFunction(&encodeFn{
		encoder: encoder,
	})
	svc.AddFunction(&decodeFn{
		encoder: encoder,
	})
	return
}

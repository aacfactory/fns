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
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/services"
)

const (
	name     = "authorizations"
	formatFn = "format"
	parseFn  = "parse"
)

var (
	ErrUnauthorized = errors.Unauthorized("fns: unauthorized")
)

func Service(encoder TokenEncoder) (v services.Service) {
	if encoder == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: service requires token encoder component").WithMeta("service", name)))
		return
	}
	v = &service{
		Abstract: services.NewAbstract(name, true, encoder),
	}
	return
}

type service struct {
	services.Abstract
	encoder TokenEncoder
}

func (svc *service) Construct(options services.Options) (err error) {
	err = svc.Abstract.Construct(options)
	if err != nil {
		return
	}
	if svc.Components() == nil || len(svc.Components()) != 1 {
		err = errors.Warning("fns: build failed").WithMeta("service", name).WithCause(errors.Warning("fns: token encoder is required"))
		return
	}
	for _, component := range svc.Components() {
		encoder, ok := component.(TokenEncoder)
		if !ok {
			err = errors.Warning("fns: build failed").WithMeta("service", name).WithCause(errors.Warning("fns: token encoder is required"))
			return
		}
		svc.encoder = encoder
	}
	return
}

func (svc *service) Handle(ctx context.Context, fn string, argument services.Argument) (v interface{}, err error) {
	switch fn {
	case formatFn:

		break
	case parseFn:

		break
	default:
		err = errors.Warning("authorizations: fn was not found")
		break
	}
	return
}

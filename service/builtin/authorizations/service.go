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
	"github.com/aacfactory/fns/service"
)

const (
	createFn = "create"
	verifyFn = "verify"
)

func Service(tokens Tokens) (v service.Service) {
	if tokens == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("authorizations: service requires tokens component")))
		return
	}
	v = &_service_{
		Abstract: service.NewAbstract("authorizations", true, tokens),
	}
	return
}

type _service_ struct {
	service.Abstract
	tokens Tokens
}

func (svc *_service_) Build(options service.Options) (err error) {
	err = svc.Abstract.Build(options)
	if err != nil {
		return
	}
	if svc.Components() == nil || len(svc.Components()) != 1 {
		err = errors.Warning("authorizations: build failed").WithCause(errors.Warning("authorizations: tokens is required"))
		return
	}
	for _, component := range svc.Components() {
		tokens, ok := component.(Tokens)
		if !ok {
			err = errors.Warning("authorizations: build failed").WithCause(errors.Warning("authorizations: tokens is required"))
			return
		}
		svc.tokens = tokens
	}
	return
}

func (svc *_service_) Document() (doc service.Document) {
	return
}

func (svc *_service_) Handle(ctx context.Context, fn string, argument service.Argument) (v interface{}, err errors.CodeError) {
	switch fn {
	case createFn:
		param := CreateTokenParam{}
		paramErr := argument.As(&param)
		if paramErr != nil {
			err = errors.Warning("authorizations: create token failed").WithCause(paramErr)
			break
		}
		v, err = svc.tokens.Create(ctx, param)
		break
	case verifyFn:
		param := Token("")
		paramErr := argument.As(&param)
		if paramErr != nil {
			err = errors.Warning("authorizations: verify token failed").WithCause(paramErr)
			break
		}
		v, err = svc.tokens.Verify(ctx, param)
		break
	default:
		err = errors.Warning("authorizations: fn was not found")
		break
	}
	return
}

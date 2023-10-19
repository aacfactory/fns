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

func Service(tokens Tokens) (v services.Service) {
	if tokens == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("authorizations: service requires tokens component")))
		return
	}
	v = &service{
		Abstract: services.NewAbstract(name, true, tokens),
	}
	return
}

type service struct {
	services.Abstract
	tokens Tokens
}

func (svc *service) Build(options services.Options) (err error) {
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

func (svc *service) Handle(ctx context.Context, fn string, argument services.Argument) (v interface{}, err errors.CodeError) {
	switch fn {
	case formatFn:
		param := FormatTokenParam{}
		paramErr := argument.As(&param)
		if paramErr != nil {
			err = errors.Warning("authorizations: format token failed").WithCause(paramErr)
			break
		}
		v, err = svc.tokens.Format(ctx, param)
		break
	case parseFn:
		param := Token("")
		paramErr := argument.As(&param)
		if paramErr != nil {
			err = errors.Warning("authorizations: parse token failed").WithCause(paramErr)
			break
		}
		v, err = svc.tokens.Parse(ctx, param)
		break
	default:
		err = errors.Warning("authorizations: fn was not found")
		break
	}
	return
}

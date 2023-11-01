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
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/authorizations"
)

func Service(encoder TokenEncoder) (v services.Service) {
	if encoder == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: service requires token encoder component").WithMeta("service", authorizations.EndpointName)))
		return
	}
	v = &service{
		Abstract: services.NewAbstract(authorizations.EndpointName, true, encoder),
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
		err = errors.Warning("fns: construct failed").WithMeta("service", svc.Name()).WithCause(errors.Warning("fns: token encoder is required"))
		return
	}
	for _, component := range svc.Components() {
		encoder, ok := component.(TokenEncoder)
		if !ok {
			err = errors.Warning("fns: construct failed").WithMeta("service", svc.Name()).WithCause(errors.Warning("fns: token encoder is required"))
			return
		}
		svc.encoder = encoder
	}
	return
}

func (svc *service) Handle(ctx services.Request) (v interface{}, err error) {
	_, fn := ctx.Fn()
	switch bytex.ToString(fn) {
	case authorizations.EncodeFnName:
		authorization := authorizations.Authorization{}
		err = ctx.Argument().As(&authorization)
		if err != nil {
			err = errors.BadRequest("authorizations: encode failed").WithMeta("service", svc.Name()).WithMeta("fn", string(fn)).WithCause(err)
			break
		}
		token, encodeErr := svc.encoder.Encode(ctx, authorization)
		if encodeErr != nil {
			err = errors.BadRequest("authorizations: encode failed").WithMeta("service", svc.Name()).WithMeta("fn", string(fn)).WithCause(encodeErr)
			break
		}
		v = token
		break
	case authorizations.DecodeFnName:
		token := make(authorizations.Token, 0, 1)
		err = ctx.Argument().As(&token)
		if err != nil {
			err = errors.BadRequest("authorizations: decode failed").WithMeta("service", svc.Name()).WithMeta("fn", string(fn)).WithCause(err)
			break
		}
		authorization, decodeErr := svc.encoder.Decode(ctx, token)
		if decodeErr != nil {
			err = errors.BadRequest("authorizations: decode failed").WithMeta("service", svc.Name()).WithMeta("fn", string(fn)).WithCause(decodeErr)
			break
		}
		v = authorization
		break
	default:
		err = errors.NotFound("authorizations: fn was not found").WithMeta("service", svc.Name()).WithMeta("fn", string(fn))
		break
	}
	return
}

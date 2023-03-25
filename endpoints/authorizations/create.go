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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/json"
)

func NewCreateTokenParam(userId service.RequestUserId) (param *CreateTokenParam) {
	param = &CreateTokenParam{
		UserId:  userId,
		Options: json.NewObject(),
	}
	return
}

type CreateTokenParam struct {
	UserId  service.RequestUserId `json:"userId"`
	Options *json.Object          `json:"options"`
}

func (param *CreateTokenParam) AddOption(key string, value string) *CreateTokenParam {
	_ = param.Options.Put(key, value)
	return param
}

func Create(ctx context.Context, param CreateTokenParam) (token Token, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("authorizations: create token failed").WithCause(errors.Warning("authorizations: service was not deployed"))
		return
	}
	future, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, createFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	scanErr := future.Scan(&token)
	if scanErr != nil {
		err = errors.Warning("authorizations: create token failed").WithCause(scanErr)
		return
	}
	return
}

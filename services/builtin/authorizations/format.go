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
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/json"
)

func NewFormatTokenParam(userId services.RequestUserId) (param FormatTokenParam) {
	param = FormatTokenParam{
		UserId:     userId,
		Attributes: json.NewObject(),
	}
	return
}

func (param *FormatTokenParam) AddAttribute(key string, value string) FormatTokenParam {
	_ = param.Attributes.Put(key, value)
	return *param
}

func Format(ctx context.Context, param FormatTokenParam) (token Token, err errors.CodeError) {
	endpoint, hasEndpoint := services.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("authorizations: format token failed").WithCause(errors.Warning("authorizations: service was not deployed"))
		return
	}
	future, requestErr := endpoint.RequestSync(ctx, services.NewRequest(ctx, name, formatFn, services.NewArgument(param), services.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	scanErr := future.Scan(&token)
	if scanErr != nil {
		err = errors.Warning("authorizations: format token failed").WithCause(scanErr)
		return
	}
	return
}

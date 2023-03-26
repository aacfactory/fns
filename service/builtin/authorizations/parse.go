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

func ParseContext(ctx context.Context) (err errors.CodeError) {
	request, hasRequest := service.GetRequest(ctx)
	if !hasRequest {
		err = errors.Warning("authorizations: parse token failed").WithCause(fmt.Errorf("there is no request in context"))
		return
	}
	if request.User().Authenticated() {
		return
	}
	token, hasToken := request.Header().Authorization()
	if !hasToken {
		err = errors.Unauthorized("authorizations: parse token failed").WithCause(fmt.Errorf("there is no authorization in request"))
		return
	}
	result, verifyErr := Parse(ctx, Token(token))
	if verifyErr != nil {
		err = verifyErr
		return
	}
	if !result.Valid {
		err = errors.Unauthorized("authorizations: token is invalid")
		return
	}
	if !result.UserId.Exist() {
		err = errors.Warning("authorizations: parse token failed").WithCause(fmt.Errorf("there is no user id in result"))
		return
	}
	request.User().SetId(result.UserId)
	if result.Attributes != nil {
		request.User().SetAttributes(result.Attributes)
	}
	return
}

func Parse(ctx context.Context, param Token) (result ParsedToken, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("authorizations: service was not deployed"))
		return
	}
	future, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, parseFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	scanErr := future.Scan(&result)
	if scanErr != nil {
		err = errors.Warning("authorizations: parse token failed").WithCause(scanErr)
		return
	}
	return
}

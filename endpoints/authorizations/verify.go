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
	"github.com/aacfactory/json"
)

type decodeParam struct {
	Token string `json:"token"`
}

type decodeResult struct {
	Id   string       `json:"id"`
	Attr *json.Object `json:"attr"`
}

func Verify(ctx context.Context) (err errors.CodeError) {
	request, hasRequest := service.GetRequest(ctx)
	if !hasRequest {
		err = errors.Warning("authorizations: verify user authorizations failed").WithCause(fmt.Errorf("there is no request in context"))
		return
	}
	if request.User().Authenticated() {
		return
	}
	token, hasToken := request.Header().Authorization()
	if !hasToken {
		err = errors.Unauthorized("authorizations: verify user authorizations failed").WithCause(fmt.Errorf("there is no authorization in request"))
		return
	}
	endpoint, hasEndpoint := service.GetEndpoint(ctx, "authorizations")
	if !hasEndpoint {
		err = errors.Warning("authorizations: there is no authorizations in context, please deploy authorizations service")
		return
	}
	fr := endpoint.Request(ctx, "decode", service.NewArgument(&decodeParam{
		Token: token,
	}))
	result := &decodeResult{}
	_, getResultErr := fr.Get(ctx, result)
	if getResultErr != nil {
		err = getResultErr
		return
	}
	request.User().SetId(service.RequestUserId(result.Id))
	request.User().SetAttributes(result.Attr)
	return
}

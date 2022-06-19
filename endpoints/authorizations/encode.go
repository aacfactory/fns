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

type encodeParam struct {
	Id         string       `json:"id"`
	Attributes *json.Object `json:"attributes"`
}

type encodeResult struct {
	Token string `json:"token"`
}

func Encode(ctx context.Context, userId string, userAttributes *json.Object) (token string, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, "authorizations")
	if !hasEndpoint {
		err = errors.Warning("authorizations: there is no authorizations in context, please deploy authorizations service")
		return
	}
	if userId == "" {
		err = errors.Warning("authorizations: encode token failed").WithCause(fmt.Errorf("userId is empty"))
		return
	}
	if userAttributes == nil {
		userAttributes = json.NewObject()
	}
	fr := endpoint.Request(ctx, "encode", service.NewArgument(&encodeParam{
		Id:         userId,
		Attributes: userAttributes,
	}))
	result := &encodeResult{}
	_, getResultErr := fr.Get(ctx, result)
	if getResultErr != nil {
		err = getResultErr
		return
	}
	token = result.Token
	return
}

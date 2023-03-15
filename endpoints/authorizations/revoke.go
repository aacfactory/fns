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
	"github.com/aacfactory/fns/service/builtin/authorizations"
)

type revokeParam struct {
	TokenId string `json:"tokenId"`
	UserId  string `json:"userId"`
}

func Revoke(ctx context.Context, userId string, tokenId string) (err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, "authorizations")
	if !hasEndpoint {
		err = errors.Warning("authorizations: there is no authorizations in context, please deploy authorizations service")
		return
	}
	if tokenId == "" {
		err = errors.Warning("authorizations: revoke token failed").WithCause(fmt.Errorf("tokenId is empty"))
		return
	}
	_, err = endpoint.RequestSync(ctx, service.NewRequest(ctx, authorizations.Name, "revoke", service.NewArgument(&revokeParam{
		UserId:  userId,
		TokenId: tokenId,
	})))
	return
}

func RevokeUserTokens(ctx context.Context, userId string) (err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, "authorizations")
	if !hasEndpoint {
		err = errors.Warning("authorizations: there is no authorizations in context, please deploy authorizations service")
		return
	}
	if userId == "" {
		err = errors.Warning("authorizations: revoke user tokens failed").WithCause(fmt.Errorf("userId is empty"))
		return
	}
	_, err = endpoint.RequestSync(ctx, service.NewRequest(ctx, authorizations.Name, "revoke", service.NewArgument(&revokeParam{
		UserId: userId,
	})))
	return
}

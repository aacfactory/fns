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

type RevokeParam struct {
	TokenId string `json:"tokenId"`
	UserId  string `json:"userId"`
}

func revoke(ctx context.Context, param RevokeParam) (result *service.Empty, err errors.CodeError) {
	storeComponent, hasStoreComponent := service.GetComponent(ctx, "store")
	if !hasStoreComponent {
		err = errors.Warning("fns: revoke failed").WithCause(fmt.Errorf("there is no store component in context"))
		return
	}
	store, storeOk := storeComponent.(*tokenStoreComponent)
	if !storeOk {
		err = errors.Warning("fns: revoke failed").WithCause(fmt.Errorf("the encoding component in context is not *tokenStoreComponent"))
		return
	}
	if param.TokenId != "" {
		rmErr := store.Remove(ctx, param.TokenId)
		if rmErr != nil {
			err = errors.Warning("fns: revoke failed").WithCause(rmErr)
			return
		}
	}
	if param.UserId != "" {
		rmErr := store.RemoveUserTokens(ctx, param.UserId)
		if rmErr != nil {
			err = errors.Warning("fns: revoke failed").WithCause(rmErr)
			return
		}
	}
	return
}

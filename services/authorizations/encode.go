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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns"
	"github.com/aacfactory/json"
)

type EncodeParam struct {
	Id         string       `json:"id"`
	Attributes *json.Object `json:"attributes"`
}

type EncodeResult struct {
	Token string `json:"token"`
}

func encode(ctx fns.Context, param EncodeParam) (result *EncodeResult, err errors.CodeError) {
	encoding := &tokenEncodingComponent{}
	getEncodingErr := ctx.CurrentServiceComponent("encoding", encoding)
	if getEncodingErr != nil {
		err = errors.ServiceError("fns: encode failed").WithCause(getEncodingErr)
		return
	}
	store := &tokenStoreComponent{}
	getStoreErr := ctx.CurrentServiceComponent("store", store)
	if getStoreErr != nil {
		err = errors.ServiceError("fns: encode failed").WithCause(getStoreErr)
		return
	}
	token, encodeErr := encoding.Encode(param.Id, param.Attributes)
	if encodeErr != nil {
		err = errors.ServiceError("fns: encode failed").WithCause(encodeErr)
		return
	}
	saveErr := store.Save(ctx, token)
	if saveErr != nil {
		err = errors.ServiceError("fns: encode failed").WithCause(saveErr)
		return
	}
	return
}

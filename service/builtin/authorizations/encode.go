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

type EncodeParam struct {
	Id         string       `json:"id"`
	Attributes *json.Object `json:"attributes"`
}

type EncodeResult struct {
	Token string `json:"token"`
}

func encode(ctx context.Context, param EncodeParam) (result *EncodeResult, err errors.CodeError) {
	encodingComponent, hasEncodingComponent := service.GetComponent(ctx, "encoding")
	if !hasEncodingComponent {
		err = errors.Warning("fns: encode failed").WithCause(fmt.Errorf("there is no encoding component in context"))
		return
	}
	encoder, encodingOk := encodingComponent.(*tokenEncodingComponent)
	if !encodingOk {
		err = errors.Warning("fns: encode failed").WithCause(fmt.Errorf("the encoding component in context is not *tokenEncodingComponent"))
		return
	}
	token, encodeErr := encoder.Encode(param.Id, param.Attributes)
	if encodeErr != nil {
		err = errors.ServiceError("fns: encode failed").WithCause(encodeErr)
		return
	}
	storeComponent, hasStoreComponent := service.GetComponent(ctx, "store")
	if !hasStoreComponent {
		err = errors.Warning("fns: encode failed").WithCause(fmt.Errorf("there is no store component in context"))
		return
	}
	st, storeOk := storeComponent.(*tokenStoreComponent)
	if !storeOk {
		err = errors.Warning("fns: encode failed").WithCause(fmt.Errorf("the encoding component in context is not *tokenStoreComponent"))
		return
	}
	saveErr := st.Save(ctx, token)
	if saveErr != nil {
		err = errors.ServiceError("fns: encode failed").WithCause(saveErr)
		return
	}
	result = &EncodeResult{
		Token: string(token.Bytes()),
	}
	return
}

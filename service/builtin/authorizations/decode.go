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
	"time"
)

type DecodeParam struct {
	Token string `json:"token"`
}

type DecodeResult struct {
	Id   string       `json:"id"`
	Attr *json.Object `json:"attr"`
}

func decode(ctx context.Context, param DecodeParam) (result *DecodeResult, err errors.CodeError) {
	encodingComponent, hasEncodingComponent := service.GetComponent(ctx, "encoding")
	if !hasEncodingComponent {
		err = errors.Warning("fns: decode failed").WithCause(fmt.Errorf("there is no encoding component in context"))
		return
	}
	encoder, encodingOk := encodingComponent.(TokenEncodingComponent)
	if !encodingOk {
		err = errors.Warning("fns: decode failed").WithCause(fmt.Errorf("the encoding component in context is not *tokenEncodingComponent"))
		return
	}
	token, decodeErr := encoder.Decode([]byte(param.Token))
	if decodeErr != nil {
		err = errors.Warning("fns: decode failed").WithCause(decodeErr)
		return
	}
	if token.NotAfter().Before(time.Now()) {
		err = errors.Unauthorized("fns: decode failed").WithCause(fmt.Errorf("token is expired"))
		return
	}
	storeComponent, hasStoreComponent := service.GetComponent(ctx, "store")
	if !hasStoreComponent {
		err = errors.Warning("fns: decode failed").WithCause(fmt.Errorf("there is no store component in context"))
		return
	}
	st, storeOk := storeComponent.(TokenStoreComponent)
	if !storeOk {
		err = errors.Warning("fns: decode failed").WithCause(fmt.Errorf("the encoding component in context is not *tokenStoreComponent"))
		return
	}
	if !st.Exist(ctx, token.Id()) {
		err = errors.Unauthorized("fns: decode failed").WithCause(fmt.Errorf("token maybe revoked"))
		return
	}
	id, attr := token.User()
	result = &DecodeResult{
		Id:   id,
		Attr: attr,
	}
	return
}

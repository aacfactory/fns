/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package authorizations

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"time"
)

var (
	contextUserKey = []byte("authorizations")
)

func With(ctx context.Context, authorization Authorization) context.Context {
	ctx.SetUserValue(contextUserKey, authorization)
	return ctx
}

func Load(ctx context.Context) (Authorization, bool, error) {
	return context.UserValue[Authorization](ctx, contextUserKey)
}

type Authorization struct {
	Id         Id         `json:"id" avro:"id"`
	Account    Id         `json:"account" avro:"account"`
	Attributes Attributes `json:"attributes" avro:"attributes"`
	ExpireAT   time.Time  `json:"expireAT" avro:"expireAt"`
}

func (authorization Authorization) Exist() bool {
	return authorization.Id.Exist()
}

func (authorization Authorization) Validate() bool {
	return authorization.Exist() && authorization.ExpireAT.After(time.Now())
}

var ErrUnauthorized = errors.Unauthorized("unauthorized")

func Validate(ctx context.Context) (err error) {
	authorization, has, loadErr := Load(ctx)
	if loadErr != nil {
		err = ErrUnauthorized.WithCause(loadErr)
		return
	}
	if has {
		if authorization.Validate() {
			return
		}
		err = ErrUnauthorized
		return
	}

	r := services.LoadRequest(ctx)
	token := r.Header().Token()
	if len(token) == 0 {
		err = ErrUnauthorized
		return
	}
	authorization, err = Decode(ctx, token)
	if err != nil {
		err = ErrUnauthorized.WithCause(err).WithMeta("token", string(token))
		return
	}
	if !authorization.Validate() {
		err = ErrUnauthorized
		return
	}
	With(ctx, authorization)
	return
}

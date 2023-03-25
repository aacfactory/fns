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

package permissions

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
)

type EnforceParam struct {
	UserId  service.RequestUserId `json:"userId"`
	Service string                `json:"service"`
	Fn      string                `json:"fn"`
}

type Enforcer interface {
	service.Component
	Enforce(ctx context.Context, param EnforceParam) (ok bool, err errors.CodeError)
}

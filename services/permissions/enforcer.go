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

package permissions

import (
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/authorizations"
)

type EnforceParam struct {
	Account  authorizations.Id `json:"account" avro:"account"`
	Endpoint string            `json:"endpoint" avro:"endpoint"`
	Fn       string            `json:"fn" avro:"fn"`
}

type Enforcer interface {
	services.Component
	Enforce(ctx context.Context, param EnforceParam) (ok bool, err error)
}

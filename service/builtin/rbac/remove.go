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

package rbac

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"strings"
)

type RemoveArgument struct {
	Code string `json:"code"`
}

func remove(ctx context.Context, argument RemoveArgument) (err errors.CodeError) {
	code := strings.TrimSpace(argument.Code)
	if code == "" {
		err = errors.ServiceError("rbac remove failed").WithCause(fmt.Errorf("code is nil"))
		return
	}
	store := getStore(ctx)
	record, getErr := store.Role(ctx, code)
	if getErr != nil {
		err = errors.ServiceError("rbac remove failed").WithCause(getErr)
		return
	}

	children, childrenErr := children(ctx, ChildrenArgument{
		Parent:       code,
		LoadChildren: false,
	})
	if childrenErr != nil {
		err = errors.ServiceError("rbac remove failed").WithCause(childrenErr)
		return
	}
	if children != nil && len(children) > 0 {
		err = errors.ServiceError("rbac remove failed").WithCause(fmt.Errorf("can not role which has children"))
		return
	}

	removeErr := store.RemoveRole(ctx, record)
	if removeErr != nil {
		err = errors.ServiceError("rbac remove failed").WithCause(removeErr)
		return
	}
	return
}

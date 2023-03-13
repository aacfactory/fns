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

type RoleArgument struct {
	Code         string `json:"code"`
	LoadChildren bool   `json:"loadChildren"`
}

func role(ctx context.Context, argument RoleArgument) (v *Role, err errors.CodeError) {
	code := strings.TrimSpace(argument.Code)
	if code == "" {
		err = errors.Warning("rbac: get role failed").WithCause(fmt.Errorf("code is nil"))
		return
	}
	store := getStore(ctx)
	record, getErr := store.Role(ctx, code)
	if getErr != nil {
		err = errors.Warning("rbac: get role failed").WithCause(getErr)
		return
	}
	if record == nil {
		err = errors.Warning("rbac: get role failed").WithCause(fmt.Errorf("not found"))
		return
	}

	v = record.mapToRole()
	if argument.LoadChildren {
		children, childrenErr := children(ctx, ChildrenArgument{
			Parent:       record.Name,
			LoadChildren: true,
		})
		if childrenErr != nil {
			err = errors.Warning("rbac: get role failed").WithCause(childrenErr)
			return
		}
		v.Children = children
	}
	return
}

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
	"fmt"
	"github.com/aacfactory/errors"
	"strings"
)

type RemoveArgument struct {
	Name string `json:"name"`
}

func remove(ctx context.Context, argument RemoveArgument) (err errors.CodeError) {
	name := strings.TrimSpace(argument.Name)
	if name == "" {
		err = errors.ServiceError("permissions remove failed").WithCause(fmt.Errorf("name is nil"))
		return
	}
	store := getStore(ctx)
	record, getErr := store.Role(ctx, name)
	if getErr != nil {
		err = errors.ServiceError("permissions remove failed").WithCause(getErr)
		return
	}

	children, childrenErr := children(ctx, ChildrenArgument{
		Parent:       name,
		LoadChildren: false,
	})
	if childrenErr != nil {
		err = errors.ServiceError("permissions remove failed").WithCause(childrenErr)
		return
	}
	if children != nil && len(children) > 0 {
		err = errors.ServiceError("permissions remove failed").WithCause(fmt.Errorf("can not role which has children"))
		return
	}

	removeErr := store.RemoveRole(ctx, record)
	if removeErr != nil {
		err = errors.ServiceError("permissions remove failed").WithCause(removeErr)
		return
	}
	return
}

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
	"strings"
)

type ChildrenArgument struct {
	Parent       string `json:"parent"`
	LoadChildren bool   `json:"loadChildren"`
}

func children(ctx context.Context, argument ChildrenArgument) (v []*Role, err errors.CodeError) {
	parent := strings.TrimSpace(argument.Parent)
	store := getStore(ctx)
	records, getErr := store.Children(ctx, parent)
	if getErr != nil {
		err = errors.ServiceError("permissions get children of role failed").WithCause(getErr)
		return
	}
	if records == nil || len(records) == 0 {
		return
	}
	v = make([]*Role, 0, len(records))
	for _, record := range records {
		role := record.mapToRole()
		if argument.LoadChildren {
			children, childrenErr := children(ctx, ChildrenArgument{
				Parent:       record.Name,
				LoadChildren: true,
			})
			if childrenErr != nil {
				err = errors.ServiceError("permissions get children of role failed").WithCause(childrenErr)
				return
			}
			if children != nil && len(children) > 0 {
				role.Children = children
			}
		}
		v = append(v, role)
	}
	return
}

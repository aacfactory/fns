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
	"github.com/aacfactory/errors"
)

type RolesArgument struct {
	Flat bool `json:"flat"`
}

func roles(ctx context.Context, argument RolesArgument) (v []*Role, err errors.CodeError) {
	store := getStore(ctx)

	records, recordsErr := store.Roles(ctx)
	if recordsErr != nil {
		err = errors.ServiceError("permissions get roles failed").WithCause(recordsErr)
		return
	}
	v = make([]*Role, 0, 1)
	if records == nil || len(records) == 0 {
		return
	}

	if argument.Flat {
		for _, record := range records {
			v = append(v, record.mapToRole())
		}
		return
	}
	// map to tree
	// root
	for _, record := range records {
		if record.Parent == "" {
			v = append(v, record.mapToRole())
		}
	}
	loadChildren(v, records)
	return
}

func loadChildren(roles []*Role, records []*RoleRecord) {
	for _, r := range roles {
		children := make([]*Role, 0, 1)
		for _, record := range records {
			if r.Name == record.Parent {
				children = append(children, record.mapToRole())
			}
		}
		if len(children) > 0 {
			r.Children = children
			loadChildren(children, records)
		}
	}
	return
}

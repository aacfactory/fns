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

type SaveArgument struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Parent      string    `json:"parent"`
	Policies    []*Policy `json:"policies"`
}

func save(ctx context.Context, argument SaveArgument) (err errors.CodeError) {
	code := strings.TrimSpace(argument.Code)
	if code == "" {
		err = errors.Warning("rbac: save role failed").WithCause(fmt.Errorf("code is nil"))
		return
	}
	var policies []*PolicyRecord = nil
	if argument.Policies != nil && len(argument.Policies) > 0 {
		policies = make([]*PolicyRecord, 0, 1)
		for _, policy := range argument.Policies {
			object := strings.TrimSpace(policy.Object)
			if object == "" {
				continue
			}
			action := strings.TrimSpace(policy.Action)
			if action == "" {
				action = "*"
			}
			policies = append(policies, &PolicyRecord{
				Object: object,
				Action: action,
			})
		}
	}

	store := getStore(ctx)

	saveErr := store.SaveRole(ctx, &RoleRecord{
		Code:        code,
		Name:        strings.TrimSpace(argument.Name),
		Description: strings.TrimSpace(argument.Description),
		Parent:      strings.TrimSpace(argument.Parent),
		Policies:    policies,
	})

	if saveErr != nil {
		err = errors.Warning("rbac: save role failed").WithCause(saveErr)
		return
	}

	return
}

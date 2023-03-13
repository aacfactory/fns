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

type BindArgument struct {
	Subject string   `json:"subject"`
	Roles   []string `json:"roles"`
}

func bind(ctx context.Context, argument BindArgument) (err errors.CodeError) {
	subject := strings.TrimSpace(argument.Subject)
	if subject == "" {
		err = errors.Warning("rbac: subject bind roles failed").WithCause(fmt.Errorf("subject is nil"))
		return
	}
	if argument.Roles == nil || len(argument.Roles) == 0 {
		err = errors.Warning("rbac: subject bind roles failed").WithCause(fmt.Errorf("roles is nil"))
		return
	}
	store := getStore(ctx)

	records := make([]*RoleRecord, 0, 1)
	for _, role := range argument.Roles {
		record, recordErr := store.Role(ctx, strings.TrimSpace(role))
		if recordErr != nil {
			err = errors.Warning("rbac: subject bind roles failed").WithCause(recordErr)
			return
		}
		records = append(records, record)
	}
	if len(records) == 0 {
		err = errors.Warning("rbac: subject bind roles failed").WithCause(fmt.Errorf("roles is invalid"))
		return
	}

	bindErr := store.Bind(ctx, subject, records)
	if bindErr != nil {
		err = errors.Warning("rbac: subject bind roles failed").WithCause(bindErr)
		return
	}
	return
}

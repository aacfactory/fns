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

type BindsArgument struct {
	Subject string `json:"subject"`
	Flat    bool   `json:"flat"`
}

func binds(ctx context.Context, argument BindsArgument) (v []*Role, err errors.CodeError) {
	subject := strings.TrimSpace(argument.Subject)
	if subject == "" {
		err = errors.ServiceError("rbac get subject binds roles failed").WithCause(fmt.Errorf("subject is nil"))
		return
	}

	store := getStore(ctx)

	records, recordsErr := store.Binds(ctx, subject)
	if recordsErr != nil {
		err = errors.ServiceError("rbac get subject binds roles failed").WithCause(recordsErr)
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
	for _, record := range records {
		if record.Parent == "" {
			v = append(v, record.mapToRole())
		}
	}
	loadChildren(v, records)
	return
}

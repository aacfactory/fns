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

type EnforceArgument struct {
	Subject string `json:"subject"`
	Object  string `json:"object"`
	Action  string `json:"action"`
}

type EnforceResult struct {
	Pass bool `json:"pass"`
}

func enforce(ctx context.Context, argument EnforceArgument) (result *EnforceResult, err errors.CodeError) {
	subject := strings.TrimSpace(argument.Subject)
	if subject == "" {
		err = errors.ServiceError("rbac enforce failed").WithCause(fmt.Errorf("subject is nil"))
		return
	}
	object := strings.TrimSpace(argument.Object)
	if object == "" {
		err = errors.ServiceError("rbac enforce failed").WithCause(fmt.Errorf("object is nil"))
		return
	}
	action := strings.TrimSpace(argument.Action)
	if action == "" {
		err = errors.ServiceError("rbac enforce failed").WithCause(fmt.Errorf("action is nil"))
		return
	}

	roles, getBindsErr := bounds(ctx, BoundsArgument{
		Subject: subject,
		Flat:    true,
	})
	if getBindsErr != nil {
		err = errors.ServiceError("rbac enforce failed").WithCause(getBindsErr)
		return
	}
	result = &EnforceResult{
		Pass: false,
	}
	if roles == nil || len(roles) == 0 {
		return
	}
	for _, role := range roles {
		ok := role.enforce(object, action)
		if ok {
			result.Pass = true
			return
		}
	}
	return
}

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

package logs

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
)

var (
	contextKey = []byte("@fns:context:log")
)

func With(ctx context.Context, v Logger) {
	ctx.SetLocalValue(contextKey, v)
}

func Load(ctx context.Context) Logger {
	v := ctx.LocalValue(contextKey)
	if v == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: there is no log in context")))
		return nil
	}
	lg, ok := v.(Logger)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: contextKey in context is not github.com/aacfactory/logs.Logger")))
		return nil
	}
	return lg
}

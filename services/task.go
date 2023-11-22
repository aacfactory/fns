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

package services

import (
	sc "context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/futures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services/tracings"
)

type FnTask struct {
	Fn      Fn
	Promise futures.Promise
}

func (task FnTask) Execute(ctx sc.Context) {
	r := LoadRequest(context.Wrap(ctx))
	// tracing
	trace, hasTrace := tracings.Load(r)
	if hasTrace {
		trace.Waited()
	}
	v, err := task.Fn.Handle(r)
	if err != nil {
		ep, fn := r.Fn()
		codeErr := errors.Map(err).WithMeta("endpoint", bytex.ToString(ep)).WithMeta("fn", bytex.ToString(fn))
		if hasTrace {
			trace.Finish("succeed", "false", "cause", codeErr.Name())
		}
		task.Promise.Failed(codeErr)
	} else {
		if hasTrace {
			trace.Finish("succeed", "true")
		}
		task.Promise.Succeed(NewResponse(v))
	}
}

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

package services

import (
	"context"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/services/tracing"
)

type traceReportTask struct {
	endpoint Endpoint
	tracer   tracing.Tracer
}

func (task traceReportTask) Execute(ctx context.Context) {
	req := AcquireRequest(
		ctx,
		bytex.FromString(tracing.ServiceName), bytex.FromString(tracing.ReportFnName),
		NewArgument(task.tracer),
		WithInternalRequest(),
	)
	_, _ = task.endpoint.Handle(req)
	ReleaseRequest(req)
}

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

package metrics

import (
	sc "context"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"sync/atomic"
)

type ReporterComponent struct {
	reporter Reporter
}

func (c *ReporterComponent) Name() (name string) {
	name = "reporter"
	return
}

func (c *ReporterComponent) Construct(options services.Options) (err error) {
	err = c.reporter.Construct(options)
	return
}

func (c *ReporterComponent) Shutdown(ctx context.Context) {
	c.reporter.Shutdown(ctx)
}

func (c *ReporterComponent) Report(ctx context.Context, metric Metric) {
	c.reporter.Report(ctx, metric)
}

type Reporter interface {
	Construct(options services.Options) (err error)
	Shutdown(ctx context.Context)
	Report(ctx context.Context, metric Metric)
}

var (
	endpointStatus = atomic.Int64{}
)

type ReportTask struct {
	rt     *runtime.Runtime
	metric Metric
}

func (task *ReportTask) Name() (name string) {
	name = "metric"
	return
}

func (task *ReportTask) Execute(_ sc.Context) {
	rt := task.rt
	ctx := runtime.With(context.TODO(), rt)
	eps := rt.Endpoints()
	if endpointStatus.Load() < 5 {
		_, has := eps.Get(ctx, endpointName)
		if has {
			endpointStatus.Store(5)
		} else {
			endpointStatus.Add(1)
		}
	}
	if endpointStatus.Load() > 4 {
		_, _ = eps.Request(ctx, endpointName, reportFnName, task.metric, services.WithInternalRequest())
	}
	return
}

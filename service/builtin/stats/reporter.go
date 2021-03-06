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

package stats

import (
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/logs"
)

type ReporterOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Reporter interface {
	Build(options ReporterOptions) (err error)
	Report(ctx context.Context, metric *Metric) (err error)
	Close()
}

type reporterComponent struct {
	reporter Reporter
}

func (component *reporterComponent) Name() (name string) {
	name = "reporter"
	return
}

func (component *reporterComponent) Build(options service.ComponentOptions) (err error) {
	config, hasConfig := options.Config.Node("reporter")
	if !hasConfig {
		err = errors.Warning("fns: build stats reporter failed").WithCause(fmt.Errorf("there is no reporter node in stats config node"))
		return
	}
	err = component.reporter.Build(ReporterOptions{
		Log:    options.Log,
		Config: config,
	})
	if err != nil {
		err = errors.Warning("fns: build stats reporter failed").WithCause(err)
		return
	}
	return
}

func (component *reporterComponent) Close() {
	component.reporter.Close()
}

func getReporter(ctx context.Context) (v Reporter) {
	vv, has := service.GetComponent(ctx, "reporter")
	if !has {
		panic(errors.Warning("fns: get stats reporter failed").WithCause(fmt.Errorf("reporter was not found in context")))
	}
	v = vv.(*reporterComponent).reporter
	return
}

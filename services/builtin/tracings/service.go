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

package tracings

import (
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/tracing"
	"github.com/aacfactory/fns/services/validators"
)

func Service(reporter Reporter) (v services.Service) {
	if reporter == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("tracings: create tracings service failed").WithCause(fmt.Errorf("reporter is nil"))))
	}
	v = &service{
		Abstract: services.NewAbstract(tracing.EndpointName, true, reporter),
	}
	return
}

type service struct {
	services.Abstract
	reporter Reporter
}

func (svc *service) Construct(options services.Options) (err error) {
	err = svc.Abstract.Construct(options)
	if err != nil {
		return
	}
	if svc.Components() == nil || len(svc.Components()) != 1 {
		err = errors.Warning("tracings: construct failed").WithMeta("endpoint", svc.Name()).WithCause(errors.Warning("tracings: reporter is required"))
		return
	}
	for _, component := range svc.Components() {
		reporter, ok := component.(Reporter)
		if !ok {
			err = errors.Warning("tracings: construct failed").WithMeta("endpoint", svc.Name()).WithCause(errors.Warning("tracings: reporter is required"))
			return
		}
		svc.reporter = reporter
	}
	return
}

func (svc *service) Handle(ctx services.Request) (v interface{}, err error) {
	_, fn := ctx.Fn()
	switch bytex.ToString(fn) {
	case tracing.ReportFnName:
		tracer := tracing.Tracer{}
		paramErr := ctx.Argument().As(&tracer)
		if paramErr != nil {
			err = errors.Warning("tracings: decode param failed").WithCause(paramErr).WithMeta("service", svc.Name()).WithMeta("fn", string(fn))
			break
		}
		validErr := validators.Validate(tracer)
		if validErr != nil {
			err = validErr.WithMeta("service", svc.Name()).WithMeta("fn", string(fn))
			break
		}
		err = svc.reporter.Report(ctx, tracer)
		if err != nil {
			err = errors.Warning("tracings: report failed").WithMeta("service", svc.Name()).WithMeta("fn", string(fn)).WithCause(err)
		}
		break
	default:
		err = errors.NotFound("tracings: fn was not found").WithMeta("fn", string(fn))
		break
	}
	return
}

func (svc *service) Close() {}

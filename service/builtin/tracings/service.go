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
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/fns/service/validators"
)

const (
	name = "tracings"
)

func Service(reporter Reporter) (v service.Service) {
	if reporter == nil {
		panic(errors.Warning("fns: create tracings service failed").WithCause(fmt.Errorf("reporter is nil")))
	}
	v = &tracing{
		components: map[string]service.Component{"reporter": &reporterComponent{reporter: reporter}},
	}
	return
}

type tracing struct {
	components map[string]service.Component
}

func (svc *tracing) Build(options service.Options) (err error) {
	if svc.components != nil {
		for cn, component := range svc.components {
			if component == nil {
				continue
			}
			componentCfg, hasConfig := options.Config.Node(cn)
			if !hasConfig {
				componentCfg, _ = configures.NewJsonConfig([]byte("{}"))
			}
			err = component.Build(service.ComponentOptions{
				Log:    options.Log.With("component", cn),
				Config: componentCfg,
			})
			if err != nil {
				err = errors.Warning("fns: build tracings service failed").WithCause(err)
				return
			}
		}
	}
	return
}

func (svc *tracing) Name() string {
	return name
}

func (svc *tracing) Internal() bool {
	return true
}

func (svc *tracing) Components() (components map[string]service.Component) {
	components = svc.components
	return
}

func (svc *tracing) Document() (doc service.Document) {
	return
}

func (svc *tracing) Handle(context context.Context, fn string, argument service.Argument) (v interface{}, err errors.CodeError) {
	switch fn {
	case "report":
		tracer := &Tracer{}
		asErr := argument.As(tracer)
		if asErr != nil {
			err = errors.BadRequest("fns: decode argument failed").WithCause(asErr).WithMeta("service", name).WithMeta("fn", fn)
			break
		}
		validErr := validators.Validate(tracer)
		if validErr != nil {
			err = validErr.WithMeta("service", name).WithMeta("fn", fn)
			break
		}
		err = report(context, tracer)
		if err != nil {
			err = err.WithMeta("service", name).WithMeta("fn", fn)
		}
		break
	default:
		err = errors.NotFound("fns: fn was not found").WithMeta("fn", fn)
		break
	}
	return
}

func (svc *tracing) Close() {

}

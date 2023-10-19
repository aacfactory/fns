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
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/services/validators"
)

const (
	name = "stats"
)

func Service(components ...services.Component) (v services.Service) {
	var reporter services.Component
	for _, component := range components {
		if component.Name() == "reporter" {
			reporter = component
			continue
		}
		if reporter != nil {
			break
		}
	}
	if reporter == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("stats: create stats service failed").WithCause(fmt.Errorf("reporter is nil"))))
	}
	v = &service{
		components: map[string]services.Component{"reporter": reporter},
	}
	return
}

type service struct {
	components map[string]services.Component
}

func (svc *service) Build(options services.Options) (err error) {
	if svc.components != nil {
		for cn, component := range svc.components {
			if component == nil {
				continue
			}
			componentCfg, hasConfig := options.Config.Node(cn)
			if !hasConfig {
				componentCfg, _ = configures.NewJsonConfig([]byte("{}"))
			}
			err = component.Build(services.ComponentOptions{
				Log:    options.Log.With("component", cn),
				Config: componentCfg,
			})
			if err != nil {
				err = errors.Warning("stats: build stats service failed").WithCause(err)
				return
			}
		}
	}
	return
}

func (svc *service) Name() string {
	return name
}

func (svc *service) Internal() bool {
	return true
}

func (svc *service) Components() (components map[string]services.Component) {
	components = svc.components
	return
}

func (svc *service) Document() (doc *documents.Document) {
	return
}

func (svc *service) Handle(context context.Context, fn string, argument services.Argument) (v interface{}, err errors.CodeError) {
	switch fn {
	case "report":
		metric := &Metric{}
		asErr := argument.As(metric)
		if asErr != nil {
			err = errors.Warning("stats: decode argument failed").WithCause(asErr).WithMeta("service", name).WithMeta("fn", fn)
			break
		}
		validErr := validators.Validate(metric)
		if validErr != nil {
			err = validErr.WithMeta("service", name).WithMeta("fn", fn)
			break
		}
		err = report(context, metric)
		if err != nil {
			err = err.WithMeta("service", name).WithMeta("fn", fn)
		}
		break
	default:
		err = errors.Warning("stats: fn was not found").WithMeta("service", name).WithMeta("fn", fn)
		break
	}
	return
}

func (svc *service) Close() {}

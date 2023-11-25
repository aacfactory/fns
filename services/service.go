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
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/logs"
	"sort"
)

type Options struct {
	Id      string
	Version versions.Version
	Log     logs.Logger
	Config  configures.Config
}

type Service interface {
	Endpoint
	Construct(options Options) (err error)
	Components() (components Components)
}

type Listenable interface {
	Service
	// Listen
	// ctx with runtime
	Listen(ctx context.Context) (err error)
}

func NewAbstract(name string, internal bool, components ...Component) Abstract {
	svc := Abstract{
		name:       name,
		internal:   internal,
		log:        nil,
		components: make(Components, 0, 1),
		functions:  make(Fns, 0, 1),
	}
	if components != nil && len(components) > 0 {
		for _, component := range components {
			if component == nil {
				continue
			}
			svc.components = append(svc.components, component)
		}
	}
	return svc
}

type Abstract struct {
	id         string
	name       string
	version    versions.Version
	internal   bool
	log        logs.Logger
	components Components
	functions  Fns
}

func (abstract *Abstract) Construct(options Options) (err error) {
	abstract.log = options.Log
	abstract.id = options.Id
	abstract.version = options.Version
	if abstract.components != nil {
		for _, component := range abstract.components {
			config, hasConfig := options.Config.Node(component.Name())
			if !hasConfig {
				config, _ = configures.NewJsonConfig([]byte{'{', '}'})
			}
			constructErr := component.Construct(Options{
				Id:      abstract.id,
				Version: abstract.version,
				Log:     abstract.log.With("component", component.Name()),
				Config:  config,
			})
			if constructErr != nil {
				if abstract.log.ErrorEnabled() {
					abstract.log.Error().Caller().Cause(errors.Map(constructErr).WithMeta("component", component.Name())).Message("service: construct component failed")
				}
				err = errors.Warning(fmt.Sprintf("fns: %s construct failed", abstract.name)).WithMeta("service", abstract.name).WithCause(constructErr)
				return
			}
			return
		}
	}
	return
}

func (abstract *Abstract) Id() string {
	return abstract.id
}

func (abstract *Abstract) Name() (name string) {
	name = abstract.name
	return
}

func (abstract *Abstract) Version() versions.Version {
	return abstract.version
}

func (abstract *Abstract) Internal() (internal bool) {
	internal = abstract.internal
	return
}

func (abstract *Abstract) Components() (components Components) {
	components = abstract.components
	return
}

func (abstract *Abstract) Document() (document documents.Endpoint) {
	return
}

func (abstract *Abstract) AddFunction(fn Fn) {
	abstract.functions = abstract.functions.Add(fn)
	sort.Sort(abstract.functions)
}

func (abstract *Abstract) Functions() (functions Fns) {
	functions = abstract.functions
	return
}

func (abstract *Abstract) Shutdown(ctx context.Context) {
	if abstract.components != nil && len(abstract.components) > 0 {
		for _, component := range abstract.components {
			component.Shutdown(ctx)
		}
	}
	if abstract.log.DebugEnabled() {
		abstract.log.Debug().Message(fmt.Sprintf("%s: closed", abstract.name))
	}
	return
}

func (abstract *Abstract) Log() (log logs.Logger) {
	log = abstract.log
	return
}

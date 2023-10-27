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
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/logs"
)

type Options struct {
	Log    logs.Logger
	Config configures.Config
}

const (
	contextComponentsKey = "@fns:service:components"
)

type Component interface {
	Name() (name string)
	Construct(options Options) (err error)
	Close()
}

type Components map[string]Component

func WithComponents(ctx context.Context, components Components) context.Context {
	ctx = context.WithValue(ctx, contextComponentsKey, components)
	return ctx
}

func LoadComponents(ctx context.Context) Components {
	v := ctx.Value(contextComponentsKey)
	if v == nil {
		return nil
	}
	c, ok := v.(Components)
	if !ok {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: components in context is not github.com/aacfactory/fns/services.Components")))
		return nil
	}
	return c
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
		components: make(map[string]Component),
	}
	if components != nil && len(components) > 0 {
		for _, component := range components {
			if component == nil {
				continue
			}
			svc.components[component.Name()] = component
		}
	}
	return svc
}

type Abstract struct {
	name       string
	internal   bool
	log        logs.Logger
	components map[string]Component
}

func (abstract *Abstract) Construct(options Options) (err error) {
	abstract.log = options.Log
	if abstract.components != nil {
		for _, component := range abstract.components {
			config, hasConfig := options.Config.Node(component.Name())
			if !hasConfig {
				config, _ = configures.NewJsonConfig([]byte{'{', '}'})
			}
			constructErr := component.Construct(Options{
				Log:    abstract.log.With("component", component.Name()),
				Config: config,
			})
			if constructErr != nil {
				if abstract.log.ErrorEnabled() {
					abstract.log.Error().Caller().Cause(errors.Map(constructErr).WithMeta("component", component.Name())).Message("service: construct component failed")
				}
				err = errors.Warning(fmt.Sprintf("%s: build failed", abstract.name)).WithMeta("service", abstract.name).WithCause(constructErr)
			}
			return
		}
	}
	return
}

func (abstract *Abstract) Name() (name string) {
	name = abstract.name
	return
}

func (abstract *Abstract) Internal() (internal bool) {
	internal = abstract.internal
	return
}

func (abstract *Abstract) Components() (components Components) {
	components = abstract.components
	return
}

func (abstract *Abstract) Document() (document *documents.Document) {
	return
}

func (abstract *Abstract) Close() {
	if abstract.components != nil && len(abstract.components) > 0 {
		for _, component := range abstract.components {
			component.Close()
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

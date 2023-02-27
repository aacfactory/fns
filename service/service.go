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

package service

import (
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
)

type ComponentOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Component interface {
	Name() (name string)
	Build(options ComponentOptions) (err error)
	Close()
}

type Options struct {
	Log    logs.Logger
	Config configures.Config
}

// Service
// 管理 Fn 的服务
type Service interface {
	Build(options Options) (err error)
	Name() (name string)
	Internal() (internal bool)
	Components() (components map[string]Component)
	Document() (doc Document)
	Handle(context context.Context, fn string, argument Argument) (v interface{}, err errors.CodeError)
	Close()
}

type Listenable interface {
	Service
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

func (svc *Abstract) Build(options Options) (err error) {
	svc.log = options.Log
	if svc.components != nil {
		for _, component := range svc.components {
			componentConfig, hasComponentConfig := options.Config.Node(component.Name())
			if !hasComponentConfig {
				componentConfig, _ = configures.NewJsonConfig([]byte{'{', '}'})
			}
			componentBuildErr := component.Build(ComponentOptions{
				Log:    svc.log.With("component", component.Name()),
				Config: componentConfig,
			})
			if componentBuildErr != nil {
				if svc.log.ErrorEnabled() {
					svc.log.Error().Caller().Cause(errors.Map(componentBuildErr).WithMeta("component", component.Name())).Message("service: build component failed")
				}
				err = errors.Warning(fmt.Sprintf("%s: build failed", svc.name)).WithMeta("service", svc.name).WithCause(componentBuildErr)
			}
			return
		}
	}
	return
}

func (svc *Abstract) Name() (name string) {
	name = svc.name
	return
}

func (svc *Abstract) Internal() (internal bool) {
	internal = svc.internal
	return
}

func (svc *Abstract) Components() (components map[string]Component) {
	components = svc.components
	return
}

func (svc *Abstract) Close() {
	if svc.components != nil && len(svc.components) > 0 {
		for _, component := range svc.components {
			component.Close()
		}
	}
	if svc.log.DebugEnabled() {
		svc.log.Debug().Message(fmt.Sprintf("%s: closed", svc.name))
	}
	return
}

func (svc *Abstract) Log() (log logs.Logger) {
	log = svc.log
	return
}

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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
)

var (
	contextComponentsKeyPrefix = []byte("@fns:service:components:")
)

type Component interface {
	Name() (name string)
	Construct(options Options) (err error)
	Shutdown(ctx context.Context)
}

type Components []Component

func (components Components) Get(name string) (v Component, has bool) {
	for _, component := range components {
		if component.Name() == name {
			v = component
			has = true
			return
		}
	}
	return
}

func WithComponents(ctx context.Context, service []byte, components Components) {
	ctx.SetLocalValue(append(contextComponentsKeyPrefix, service...), components)
}

func LoadComponents(ctx context.Context, service []byte) Components {
	v := ctx.LocalValue(append(contextComponentsKeyPrefix, service...))
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

func LoadComponent[C Component](ctx context.Context, service []byte, name string) (c C, has bool) {
	cc := LoadComponents(ctx, service)
	if len(cc) == 0 {
		return
	}
	v, exist := cc.Get(name)
	if !exist {
		return
	}
	c, has = v.(C)
	return
}

func GetComponent[C Component](ctx context.Context, name string) (c C, has bool) {
	req := LoadRequest(ctx)
	service, _ := req.Fn()
	c, has = LoadComponent[C](ctx, service, name)
	return
}

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
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
)

type ComponentOptions struct {
	Log    logs.Logger
	Config configuares.Config
}

type Component interface {
	Name() (name string)
	Build(options ComponentOptions) (err error)
}

type Options struct {
	Log    logs.Logger
	Config configuares.Config
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

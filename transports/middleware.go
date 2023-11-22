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

package transports

import (
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
	"strings"
)

type MiddlewareOptions struct {
	Log    logs.Logger
	Config configures.Config
}

type Middleware interface {
	Name() string
	Construct(options MiddlewareOptions) error
	Handler(next Handler) Handler
	Close()
}

func WaveMiddlewares(log logs.Logger, config Config, middlewares []Middleware) (v Middlewares, err error) {
	for _, middleware := range middlewares {
		name := strings.TrimSpace(middleware.Name())
		mc, mcErr := config.MiddlewareConfig(name)
		if mcErr != nil {
			err = errors.Warning("wave middlewares failed").WithCause(mcErr).WithMeta("middleware", name)
			return
		}
		constructErr := middleware.Construct(MiddlewareOptions{
			Log:    log.With("middleware", name),
			Config: mc,
		})
		if constructErr != nil {
			err = errors.Warning("wave middlewares failed").WithCause(constructErr).WithMeta("middleware", name)
			return
		}
	}
	v = middlewares
	return
}

type Middlewares []Middleware

func (middlewares Middlewares) Handler(handler Handler) Handler {
	if len(middlewares) == 0 {
		return handler
	}
	for i := len(middlewares) - 1; i > -1; i-- {
		handler = middlewares[i].Handler(handler)
	}
	return handler
}

func (middlewares Middlewares) Close() {
	for _, middleware := range middlewares {
		middleware.Close()
	}
}

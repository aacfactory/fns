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

package commons

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/authorizations"
	"github.com/aacfactory/fns/services/metrics"
	"github.com/aacfactory/fns/services/permissions"
	"reflect"
)

var (
	nilType = reflect.TypeOf(new(NIL))
)

type NIL struct{}

type FnOptions struct {
	readonly      bool
	internal      bool
	deprecated    bool
	authorization bool
	permission    bool
	cacheMod      []string
	cacheControl  []string
	metric        bool
	barrier       bool
}

// todo use options
// add validate cache cache-control
func NewFn[P any, R any](name string, readonly bool, internal bool, authorization bool, permission bool, metric bool, barrier bool, handler FnHandler, middlewares ...FnHandlerMiddleware) services.Fn {
	if len(middlewares) > 0 {
		handler = FnHandlerMiddlewares(middlewares).Handler(handler)
	}
	return &Fn[R]{
		name:          name,
		internal:      internal,
		readonly:      readonly,
		authorization: authorization,
		permission:    permission,
		metric:        metric,
		barrier:       barrier,
		handler:       handler,
		hasParam:      reflect.TypeOf(new(P)) != nilType,
		hasResult:     reflect.TypeOf(new(R)) != nilType,
	}
}

type FnHandler func(ctx services.Request) (v any, err error)

type FnHandlerMiddleware interface {
	Handler(next FnHandler) FnHandler
}

type FnHandlerMiddlewares []FnHandlerMiddleware

func (middlewares FnHandlerMiddlewares) Handler(handler FnHandler) FnHandler {
	if len(middlewares) == 0 {
		return handler
	}
	for i := len(middlewares) - 1; i > -1; i-- {
		handler = middlewares[i].Handler(handler)
	}
	return handler
}

// Fn
// builtin fn handler wrapper
// supported annotations
// @fn {name}
// @readonly
// @authorization
// @permission
// @validation
// @cache {get} {set} {get-set} {remove} {ttl}
// @cache-control {max-age=sec} {public=true} {must-revalidate} {proxy-revalidate}
// @barrier
// @metric
// @middlewares >>>
// {path}.{IdentName}
// ...
// <<<
// @title {title}
// @description >>>
// {description}
// <<<
// @errors >>>
// {error_name}
// zh: {zh_message}
// en: {en_message}
// <<<
type Fn[R any] struct {
	name          string
	internal      bool
	readonly      bool
	authorization bool
	permission    bool
	metric        bool
	barrier       bool
	handler       FnHandler
	hasParam      bool
	hasResult     bool
}

func (fn *Fn[R]) Name() string {
	return fn.name
}

func (fn *Fn[R]) Internal() bool {
	return fn.internal
}

func (fn *Fn[R]) Readonly() bool {
	return fn.readonly
}

func (fn *Fn[R]) Handle(r services.Request) (v interface{}, err error) {
	if fn.internal && !r.Header().Internal() {
		err = errors.NotAcceptable("fns: fn cannot be accessed externally")
		return
	}
	if fn.barrier {
		var key []byte
		if fn.authorization {
			key, err = services.HashRequest(r, services.HashRequestWithToken())
		} else {
			key, err = services.HashRequest(r)
		}
		if err != nil {
			return
		}
		if fn.metric {
			metrics.Begin(r)
		}
		obj, doErr := runtime.Barrier(r, key, func() (result interface{}, err error) {
			// authorization
			if fn.authorization {
				err = fn.verifyAuthorization(r)
				if err != nil {
					return
				}
			}
			// permission
			if fn.permission {
				err = fn.verifyPermission(r)
				if err != nil {
					return
				}
			}
			// handle
			result, err = fn.handler(r)
			return
		})
		if doErr == nil && fn.hasResult {
			v, err = services.ValueOfResponse[R](obj)
		} else {
			err = doErr
		}
		if fn.metric {
			if err != nil {
				metrics.EndWithCause(r, err)
			} else {
				metrics.End(r)
			}
		}
	} else {
		if fn.metric {
			metrics.Begin(r)
		}
		// authorization
		if fn.authorization {
			err = fn.verifyAuthorization(r)
			if err != nil {
				if fn.metric {
					metrics.EndWithCause(r, err)
				}
				return
			}
		}
		// permission
		if fn.permission {
			err = fn.verifyPermission(r)
			if err != nil {
				if fn.metric {
					metrics.EndWithCause(r, err)
				}
				return
			}
		}
		// handle
		v, err = fn.handler(r)
		if fn.metric {
			if err != nil {
				metrics.EndWithCause(r, err)
			} else {
				metrics.End(r)
			}
		}
	}
	return
}

func (fn *Fn[R]) verifyAuthorization(r services.Request) (err error) {
	authorization, has, loadErr := authorizations.Load(r)
	if loadErr != nil {
		err = authorizations.ErrUnauthorized.WithCause(loadErr)
		return
	}
	if !has {
		err = authorizations.ErrUnauthorized
		return
	}
	if authorization.Exist() {
		if !authorization.Validate() {
			err = authorizations.ErrUnauthorized
			return
		}
	} else {
		token := r.Header().Token()
		if len(token) == 0 {
			err = authorizations.ErrUnauthorized
			return
		}
		authorization, err = authorizations.Decode(r, token)
		if err != nil {
			err = authorizations.ErrUnauthorized.WithCause(err)
			return
		}
		authorizations.With(r, authorization)
	}
	return
}

func (fn *Fn[R]) verifyPermission(r services.Request) (err error) {
	err = permissions.EnforceContext(r)
	return
}

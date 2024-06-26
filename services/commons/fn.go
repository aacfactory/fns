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
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/logs"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/authorizations"
	"github.com/aacfactory/fns/services/caches"
	"github.com/aacfactory/fns/services/metrics"
	"github.com/aacfactory/fns/services/permissions"
	"github.com/aacfactory/fns/services/validators"
	"github.com/aacfactory/fns/transports/middlewares/cachecontrol"
	"reflect"
	"strconv"
	"time"
)

var (
	emptyType = reflect.TypeOf(new(services.Empty))
)

type FnHandler[P any, R any] func(ctx context.Context, param P) (v R, err error)

type FnOptions struct {
	readonly        bool
	internal        bool
	deprecated      bool
	validation      bool
	validationTitle string
	authorization   bool
	permission      bool
	cacheCommand    string
	cacheTTL        time.Duration
	cacheControl    []cachecontrol.MakeOption
	metric          bool
	barrier         bool
}

type FnOption func(opt *FnOptions) (err error)

func Readonly() FnOption {
	return func(opt *FnOptions) (err error) {
		opt.readonly = true
		return
	}
}

func Internal() FnOption {
	return func(opt *FnOptions) (err error) {
		opt.internal = true
		return
	}
}

func Deprecated() FnOption {
	return func(opt *FnOptions) (err error) {
		opt.deprecated = true
		return
	}
}

func Validation(title string) FnOption {
	return func(opt *FnOptions) (err error) {
		opt.validation = true
		if title == "" {
			title = "invalid"
		}
		opt.validationTitle = title
		return
	}
}

func Authorization() FnOption {
	return func(opt *FnOptions) (err error) {
		opt.authorization = true
		return
	}
}

func Permission() FnOption {
	return func(opt *FnOptions) (err error) {
		opt.permission = true
		return
	}
}

func Metric() FnOption {
	return func(opt *FnOptions) (err error) {
		opt.metric = true
		return
	}
}

func Barrier() FnOption {
	return func(opt *FnOptions) (err error) {
		opt.barrier = true
		return
	}
}

const (
	GetCacheMod    = "get"
	GetSetCacheMod = "get-set"
	SetCacheMod    = "set"
	RemoveCacheMod = "remove"
)

func Cache(mod string, param string) FnOption {
	return func(opt *FnOptions) (err error) {
		switch mod {
		case GetCacheMod:
			opt.cacheCommand = GetCacheMod
			break
		case GetSetCacheMod:
			if param == "" {
				param = "60"
			}
			sec, secErr := strconv.ParseInt(param, 10, 64)
			if secErr != nil {
				err = errors.Warning("invalid cache param")
				break
			}
			if sec < 1 {
				sec = 60
			}
			opt.cacheCommand = GetSetCacheMod
			opt.cacheTTL = time.Duration(sec) * time.Second
			break
		case SetCacheMod:
			if param == "" {
				param = "60"
			}
			sec, secErr := strconv.ParseInt(param, 10, 64)
			if secErr != nil {
				err = errors.Warning("invalid cache param")
				break
			}
			if sec < 1 {
				sec = 60
			}
			opt.cacheCommand = SetCacheMod
			opt.cacheTTL = time.Duration(sec) * time.Second
			break
		case RemoveCacheMod:
			opt.cacheCommand = RemoveCacheMod
			break
		default:
			err = errors.Warning("invalid cache mod")
			break
		}
		return
	}
}

func CacheControl(maxAge int, public bool, mustRevalidate bool, proxyRevalidate bool) FnOption {
	return func(opt *FnOptions) (err error) {
		if maxAge > 0 {
			opt.cacheControl = append(opt.cacheControl, cachecontrol.MaxAge(maxAge))
			if public {
				opt.cacheControl = append(opt.cacheControl, cachecontrol.Public())
			}
			if mustRevalidate {
				opt.cacheControl = append(opt.cacheControl, cachecontrol.MustRevalidate())
			}
			if proxyRevalidate {
				opt.cacheControl = append(opt.cacheControl, cachecontrol.ProxyRevalidate())
			}
		}
		return
	}
}

func NewFn[P any, R any](name string, handler FnHandler[P, R], options ...FnOption) services.Fn {
	opt := FnOptions{}
	for _, option := range options {
		if optErr := option(&opt); optErr != nil {
			panic(fmt.Sprintf("%+v", errors.Warning("new fn failed").WithMeta("fn", name).WithCause(optErr)))
			return nil
		}
	}
	return &Fn[P, R]{
		name:                    name,
		internal:                opt.internal,
		readonly:                opt.readonly,
		deprecated:              opt.deprecated,
		validation:              opt.validation,
		validationTitle:         opt.validationTitle,
		authorization:           opt.authorization,
		permission:              opt.permission,
		metric:                  opt.metric,
		barrier:                 opt.barrier,
		cacheCommand:            opt.cacheCommand,
		cacheTTL:                opt.cacheTTL,
		cacheControl:            len(opt.cacheControl) > 0,
		cacheControlMakeOptions: opt.cacheControl,
		handler:                 handler,
		hasParam:                reflect.TypeOf(new(P)) != emptyType,
		hasResult:               reflect.TypeOf(new(R)) != emptyType,
	}
}

// Fn
// builtin fn handler wrapper
// supported annotations
// @fn {name}
// @internal
// @deprecated
// @readonly
// @authorization
// @permission
// @validation
// @cache {get} {set} {get-set} {remove} {ttl}
// @cache-control {max-age=sec} {public=true} {must-revalidate} {proxy-revalidate}
// @barrier
// @metric
// @title {title}
// @description >>>
// {description}
// <<<
// @errors >>>
// {error_name}
// zh: {zh_message}
// en: {en_message}
// <<<
type Fn[P any, R any] struct {
	name                    string
	internal                bool
	readonly                bool
	deprecated              bool
	authorization           bool
	permission              bool
	validation              bool
	validationTitle         string
	metric                  bool
	barrier                 bool
	cacheCommand            string
	cacheTTL                time.Duration
	cacheControl            bool
	cacheControlMakeOptions []cachecontrol.MakeOption
	handler                 FnHandler[P, R]
	hasParam                bool
	hasResult               bool
}

func (fn *Fn[P, R]) Name() string {
	return fn.name
}

func (fn *Fn[P, R]) Internal() bool {
	return fn.internal
}

func (fn *Fn[P, R]) Readonly() bool {
	return fn.readonly
}

func (fn *Fn[P, R]) Handle(r services.Request) (v interface{}, err error) {
	if fn.internal && !r.Header().Internal() {
		err = errors.NotAcceptable("fns: fn cannot be accessed externally")
		return
	}
	if fn.metric {
		metrics.Begin(r)
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
		resp, doErr := runtime.Barrier(r, key, func() (result interface{}, err error) {
			result, err = fn.handle(r)
			return
		})
		if doErr == nil && fn.hasResult {
			v, err = services.ValueOfResponse[R](resp)
		} else {
			err = doErr
		}
	} else {
		v, err = fn.handle(r)
	}
	if fn.metric {
		if err != nil {
			metrics.EndWithCause(r, err)
		} else {
			metrics.End(r)
		}
	}
	if !r.Header().Internal() {
		// cache control
		if fn.cacheControl {
			cachecontrol.Make(r, fn.cacheControlMakeOptions...)
		}
		// deprecated
		if fn.deprecated {
			services.MarkDeprecated(r)
		}
	}
	return
}

func (fn *Fn[P, R]) handle(r services.Request) (v R, err error) {
	log := logs.Load(r)
	var param P
	paramScanned := false
	// validation
	if fn.hasParam {
		if param, err = fn.param(r); err != nil {
			return
		}
		paramScanned = true
		if fn.validation {
			if err = validators.ValidateWithErrorTitle(param, fn.validationTitle); err != nil {
				return
			}
			if log.DebugEnabled() {
				log.Debug().With("validation", true).Message("fns: fn param is valid")
			}
		}
	}
	// authorization
	if fn.authorization {
		err = authorizations.ValidateContext(r)
		if err != nil {
			return
		}
		if log.DebugEnabled() {
			log.Debug().With("authorization", true).Message("fns: fn authorization is valid")
		}
	}
	// permission
	if fn.permission {
		if err = permissions.EnforceContext(r); err != nil {
			return
		}
		if log.DebugEnabled() {
			log.Debug().With("permission", true).Message("fns: fn permission is valid")
		}
	}

	// cache get or get-set
	if fn.hasParam && (fn.cacheCommand == GetCacheMod || fn.cacheCommand == GetSetCacheMod) {
		if !paramScanned {
			if param, err = fn.param(r); err != nil {
				return
			}
		}
		result, cached, cacheErr := caches.Load[R](r, param)
		if cacheErr != nil {
			if log.WarnEnabled() {
				log.Warn().Cause(cacheErr).With("fns", "caches").Message("fns: get cache failed")
			}
		}
		if log.DebugEnabled() {
			log.Debug().With("cache-hit", cached).Message("fns: get fn result from cache")
		}
		if cached {
			v = result
			return
		}
	}
	// handle
	v, err = fn.handler(r, param)
	// cache set or remove
	if fn.hasParam && fn.cacheCommand != "" {
		switch fn.cacheCommand {
		case SetCacheMod, GetSetCacheMod:
			if fn.hasResult {
				if cacheErr := caches.Set(r, param, v, fn.cacheTTL); cacheErr != nil {
					if log.WarnEnabled() {
						log.Warn().Cause(cacheErr).With("fns", "caches").Message("fns: set cache failed")
					}
				} else {
					if log.DebugEnabled() {
						log.Debug().With("cache-set", true).Message("fns: set fn result into cache succeed")
					}
				}
			}
			break
		case RemoveCacheMod:
			if cacheErr := caches.Remove(r, param); cacheErr != nil {
				if log.WarnEnabled() {
					log.Warn().Cause(cacheErr).With("fns", "caches").Message("fns: set cache failed")
				} else {
					if log.DebugEnabled() {
						log.Debug().With("cache-remove", true).Message("fns: remove fn result from cache succeed")
					}
				}
			}
			break
		default:
			break
		}
	}
	return
}

func (fn *Fn[P, R]) param(r services.Request) (param P, err error) {
	param, err = services.ValueOfParam[P](r.Param())
	if err != nil {
		err = errors.BadRequest("scan params failed").WithCause(err)
		return
	}
	return
}
